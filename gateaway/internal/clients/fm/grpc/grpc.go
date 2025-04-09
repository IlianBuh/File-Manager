package grpclient

import (
	"context"
	"errors"
	"fmt"
	filemanagerv1 "github.com/IlianBuh/fmProto/gen/go"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"io"
	"io/fs"
	"lab3/internal/lib/logger/sl"
	"log/slog"
	"path/filepath"
	"time"
)

const (
	bufsize = 4096
)

type Client struct {
	api     filemanagerv1.FileManagerClient
	log     *slog.Logger
	cnct    *grpc.ClientConn
	timeout time.Duration
}
type DataProvider interface {
	Read([]byte) (int, error)
	Close() error
}
type DataHeader interface {
	Size() int64
	Name() string
}

func New(
	log *slog.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
) (*Client, error) {
	const op = "grpclient.New"
	log.Info("creating grpc client", slog.String("op", op))

	retryOpts := []retry.CallOption{
		retry.WithCodes(codes.Aborted, codes.NotFound, codes.DeadlineExceeded),
		retry.WithMax(uint(retriesCount)),
		retry.WithPerRetryTimeout(timeout),
	}

	logOpts := []logging.Option{
		logging.WithLogOnEvents(logging.PayloadReceived, logging.PayloadSent),
	}

	cc, err := grpc.NewClient(
		"localhost:"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			logging.UnaryClientInterceptor(InterceptorLogger(log), logOpts...),
			retry.UnaryClientInterceptor(retryOpts...),
		),
	)
	if err != nil {
		log.Error("failed to connect to grpc server", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	api := filemanagerv1.NewFileManagerClient(cc)

	return &Client{
		log:     log,
		api:     api,
		cnct:    cc,
		timeout: timeout,
	}, nil
}

// InterceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func (c *Client) GetFile(ctx context.Context, filename string) ([]byte, error) {
	const op = "grpclient.GetFile"
	log := c.log.With(slog.String("op", op))
	log.Info("starting getting file from grpc server")

	if err := ctx.Err(); err != nil {
		log.Error("context return error", sl.Err(ctx.Err()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var res []byte
	stream, err := c.api.GetFile(
		ctx,
		&filemanagerv1.GetFileRequest{FileName: filename},
	)
	if err != nil {
		log.Error("failed to get file from grpc server", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for {
		recv, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			log.Error("failed to get file chunk from grpc server", sl.Err(err))
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		res = append(res, recv.GetChunk()...)
	}

	log.Info("finished getting file from grpc server",
		slog.Int("size", len(res)),
	)
	return res, nil
}

func (c *Client) DeleteFile(ctx context.Context, filename string) (err error) {
	const op = "grpclient.DeleteFile"
	log := c.log.With(slog.With("op", op))
	log.Info(
		"starting to delete file",
		slog.String("file name", filename),
	)

	if err := ctx.Err(); err != nil {
		log.Error("context return error", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	filename, _ = filepath.Localize(filename)
	_, err = c.api.DeleteFile(
		ctx,
		&filemanagerv1.DeleteFileRequest{FileName: filename},
	)
	if err != nil {
		log.Error("failed to get file from grpc server", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *Client) PostFile(ctx context.Context, data DataProvider, header DataHeader, filename string) (err error) {
	const op = "grpclient.PostFile"
	log := c.log.With(slog.String("op", op))
	log.Info("uploading file",
		slog.String("filename", header.Name()),
		slog.Int64("size", header.Size()),
	)

	if err = ctx.Err(); err != nil {
		log.Error("context return error", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	stream, err := c.api.PostFile(ctx)
	if err != nil {
		log.Error("failed to get stream from api", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if _, err := stream.CloseAndRecv(); err != nil {
			log.Error("failed to close api stream", sl.Err(err))
		}

		if err := data.Close(); err != nil {
			log.Error("failed to close data provider", sl.Err(err))
		}
	}()

	filename, _ = filepath.Localize(filename)
	if !fs.ValidPath(filename) {
		log.Warn("Invalid file path after join", slog.String("file path", filename))
		return status.Error(codes.InvalidArgument, "invalid file name")
	}

	size := header.Size()
	sent := int64(0)
	read := 0
	chunk := make([]byte, bufsize)

	for sent < size {
		read, err = data.Read(chunk)
		if err != nil {
			log.Error("failed to read file", sl.Err(err))
			return fmt.Errorf("%s: %w", op, err)
		}

		err = stream.Send(
			&filemanagerv1.PostFileRequest{
				FileName: filename,
				Chunk:    chunk[:read],
			},
		)
		if err != nil {
			log.Error("failed to send chunk", sl.Err(err))
			return fmt.Errorf("%s: %w", op, err)
		}

		sent += int64(read)
	}

	log.Info("successfully sent file")
	return nil
}

func (c *Client) PutFile(ctx context.Context, data DataProvider, header DataHeader, filename string) (err error) {
	const op = "grpclient.PutFile"
	log := c.log.With(slog.String("op", op))
	log.Info("starting to send file")

	if err = ctx.Err(); err != nil {
		log.Error("context return error", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	stream, err := c.api.PutFile(ctx)
	if err != nil {
		log.Error("failed to get stream from api", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if _, err := stream.CloseAndRecv(); err != nil {
			log.Error("failed to close api stream", sl.Err(err))
		}

		if err := data.Close(); err != nil {
			log.Error("failed to close data provider", sl.Err(err))
		}
	}()

	filename, _ = filepath.Localize(filename)
	if !fs.ValidPath(filename) {
		log.Warn("Invalid file path", slog.String("file path", filename))
		return status.Error(codes.InvalidArgument, "invalid file name")
	}

	size := header.Size()
	sent := int64(0)
	read := 0
	chunk := make([]byte, bufsize)

	for sent < size {
		read, err = data.Read(chunk)
		if err != nil {
			log.Error("failed to read file", sl.Err(err))
			return fmt.Errorf("%s: %w", op, err)
		}

		err = stream.Send(
			&filemanagerv1.PutFileRequest{
				FileName: filename,
				Chunk:    chunk[:read],
			},
		)
		if err != nil {
			log.Error("failed to send chunk", sl.Err(err))
			return fmt.Errorf("%s: %w", op, err)
		}

		sent += int64(read)
	}

	log.Info("successfully sent file")
	return nil
}

func (c *Client) Stop() {
	const op = "grpclient.Stop"

	c.log.Info("stopping grpc client")

	err := c.cnct.Close()
	if err != nil {
		c.log.Error("failed to close grpc client", sl.Err(err))
		panic(err)
	}

}
