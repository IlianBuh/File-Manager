package filemanager

import (
	"context"
	"errors"
	"fmt"
	"github.com/IlianBuh/filemanager-server/internal/lib/logger/sl"
	filemanagerv1 "github.com/IlianBuh/fmProto/gen/go"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"time"
)

type SendMessage = filemanagerv1.GetFileResponse

type Sender interface {
	MySend([]byte) error
}

type Receiver interface {
	MyReceive() (FileProvider, error)
}

type FileProvider interface {
	GetFileName() string
	GetChunk() []byte
}

type FileManager struct {
	log     *slog.Logger
	root    *os.Root
	timeout time.Duration
}

const (
	bufSize = 4096
)

func New(
	log *slog.Logger,
	rootPath string,
	timeout time.Duration,
) *FileManager {
	const op = "filemanager.New"

	root, err := os.OpenRoot(rootPath)
	if err != nil {
		panic("cannot open root directory: " + err.Error())
	}

	log.Info("created file manager",
		slog.String("root path", rootPath),
		slog.String("op", op),
	)

	return &FileManager{
		log:     log,
		root:    root,
		timeout: timeout,
	}
}

func (f *FileManager) GetFile(
	ctx context.Context,
	fileName string,
	stream Sender,
) error {
	const op = "filemanager.GetFile"
	log := f.log.With(slog.String("op", op))
	log.Info("starting to upload file", slog.String("file-name", fileName))

	if err := ctx.Err(); err != nil {
		log.Error("context error", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	stat, err := f.root.Stat(fileName)
	if err != nil {
		log.Error("failed to get stat file",
			sl.Err(err),
			slog.String("file Name: ", fileName),
		)
		return fmt.Errorf("%s: %w", op, ErrBadRequest)
	}
	if stat.IsDir() {
		log.Warn("try open directory")
		return fmt.Errorf("%s: %w", op, ErrBadRequest)
	}

	file, err := f.root.Open(fileName)
	if err != nil {
		log.Warn("failed to open file", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Error("failed to close file", sl.Err(err))
		}
	}()

	var n int
	buf := make([]byte, bufSize)
	size := int64(0)
	for {
		select {
		case <-ctx.Done():
			if err = ctx.Err(); err != nil {
				log.Error("context error", sl.Err(err))
				return fmt.Errorf("%s: %w", op, err)
			} else {
				log.Warn("context is done, failed to all file")
				return fmt.Errorf("%s: %w", op, ErrInternal)
			}
		default:
		}

		n, err = file.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Info("EOF is received")
				break
			}

			log.Error("failed to read file", sl.Err(err))
			return fmt.Errorf("%s: %w", op, err)
		}

		size += int64(n)
		err = stream.MySend(buf[:n])
		if err != nil {
			log.Error(
				"failed to send file",
				sl.Err(err),
				slog.Int64("sent", size),
			)
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	log.Info(
		"finished getting file",
		slog.Int64("sent", size),
		slog.Int64("total", stat.Size()),
	)

	return nil
}

func (f *FileManager) PostFile(
	ctx context.Context,
	recv Receiver,
) error {
	const op = "filemanager.PostFile"
	log := f.log.With(slog.String("op", op))
	log.Info("starting to download file")

	if err := ctx.Err(); err != nil {
		log.Error("context error", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	var (
		chunk      []byte
		writeCount int
		err        error
		req        FileProvider
		filepath   string
		file       *os.File
		totalSize  uint64
		success    bool = true
	)
	req, err = recv.MyReceive()
	if err != nil {
		log.Error("failed to receive file chunk", sl.Err(err))
		return fmt.Errorf("%s: %w", op, ErrReceiveFile)
	}

	filepath = req.GetFileName()
	if _, err = f.root.Stat(filepath); err == nil {
		log.Warn("trying to create file with existing file name")
		return fmt.Errorf("%s: %w", op, ErrBadRequest)
	}

	file, err = f.root.Create(filepath)
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			log.Warn("invalid file path", sl.Err(err))
			return fmt.Errorf("%s: %w", op, ErrBadRequest)
		}

		log.Error("failed to create file", sl.Err(err))
		return fmt.Errorf("%s: %w", op, ErrInternal)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Error("failed to close file", sl.Err(err))
		}
		if !success {
			if err = os.Remove(filepath); err != nil {
				log.Error("failed to remove file", sl.Err(err))
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			if err = ctx.Err(); err != nil {
				log.Error("context error", sl.Err(err))
				return fmt.Errorf("%s: %w", op, err)
			} else {
				log.Warn("context is done, failed to all file")
				return fmt.Errorf("%s: %w", op, ErrInternal)
			}
		default:
		}

		chunk = req.GetChunk()
		writeCount, err = file.Write(chunk)
		if err != nil {
			log.Error(
				"failed to write into file",
				sl.Err(err),
				slog.String("file name", filepath),
			)
			success = false
			return fmt.Errorf("%s: %w", op, ErrInternal)
		}
		totalSize += uint64(writeCount)

		req, err = recv.MyReceive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Info("end of receiving file")
				break
			}
			log.Error("failed to receive file chunk", sl.Err(err))
			success = false
			return fmt.Errorf("%s: %w", op, ErrReceiveFile)
		}

		filepath = req.GetFileName()
	}

	log.Info("successfully get file")
	return nil
}

func (f *FileManager) DeleteFile(
	ctx context.Context,
	filename string,
) error {
	const op = "filemanager.DeleteFile"
	log := f.log.With(slog.String("op", op))
	log.Info("trying to delete file", slog.String("file name", filename))

	if err := ctx.Err(); err != nil {
		log.Error("context error", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, err)
	}

	stat, err := f.root.Stat(filename)
	if err != nil {
		log.Error("failed to get file stat",
			sl.Err(err),
			slog.String("file Name: ", filename),
		)
		return fmt.Errorf("%s: %w", op, ErrBadRequest)
	}

	if stat.IsDir() {
		err = checkDirToDelete(f.log, filename)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	err = f.root.Remove(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", op, ErrInternal)
	}

	log.Info("deleted file")
	return nil
}

func (f *FileManager) PutFile(
	ctx context.Context,
	recv Receiver,
) error {
	const op = "filemanager.UpdateFile"
	log := f.log.With(slog.String("op", op))
	log.Info("starting to update file")

	if err := ctx.Err(); err != nil {
		log.Error("context error", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	var (
		chunk      []byte
		writeCount int
		err        error
		req        FileProvider
		filepath   string
		file       *os.File
		totalSize  uint64
	)
	req, err = recv.MyReceive()
	if err != nil {
		log.Error("failed to receive file chunk", sl.Err(err))
		return fmt.Errorf("%s: %w", op, ErrReceiveFile)
	}

	filepath = req.GetFileName()
	stat, err := f.root.Stat(filepath)
	if err != nil {
		log.Error("failed to get stat file",
			sl.Err(err),
			slog.String("file name: ", filepath),
		)
		return fmt.Errorf("%s: %w", op, ErrBadRequest)
	}
	if stat.IsDir() {
		log.Warn("try update directory")
		return fmt.Errorf("%s: %w", op, ErrBadRequest)
	}

	file, err = f.root.Create(filepath)
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			log.Warn("invalid file path", sl.Err(err))
			return fmt.Errorf("%s: %w", op, ErrBadRequest)
		}

		log.Error("failed to create file", sl.Err(err))
		return fmt.Errorf("%s: %w", op, ErrInternal)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error("failed to close file", sl.Err(err))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			if err = ctx.Err(); err != nil {
				log.Error("context error", sl.Err(err))
				return fmt.Errorf("%s: %w", op, err)
			} else {
				log.Warn("context is done, failed to all file")
				return fmt.Errorf("%s: %w", op, ErrInternal)
			}
		default:
		}

		chunk = req.GetChunk()
		writeCount, err = file.Write(chunk)
		if err != nil {
			log.Error(
				"failed to write into file",
				sl.Err(err),
				slog.String("file name", filepath),
			)
			return fmt.Errorf("%s: %w", op, ErrInternal)
		}
		totalSize += uint64(writeCount)

		req, err = recv.MyReceive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Info("end of receiving file")
				break
			}
			log.Error("failed to receive file chunk", sl.Err(err))
			return fmt.Errorf("%s: %w", op, ErrReceiveFile)
		}
	}

	log.Info("successfully update file")
	return nil
}

func checkDirToDelete(log *slog.Logger, filename string) error {
	const op = "filemanager.checkDirToDelete"
	log = log.With(slog.String("op", op))
	log.Info(
		"file name matched to directory",
		slog.String("file name", filename),
	)

	dir, err := os.ReadDir(filename)
	if err != nil {
		log.Error("failed to check dir",
			slog.String("dir name", filename),
			sl.Err(err),
		)
		return ErrInternal
	}

	if len(dir) > 0 {
		log.Warn("dir is not empty",
			slog.String("dir name", filename),
			sl.Err(err),
		)
		return ErrBadRequest
	}
	return nil
}
