package grpcfm

import (
	"context"
	"errors"

	"github.com/IlianBuh/filemanager-server/internal/grpc/wrappers"
	"github.com/IlianBuh/filemanager-server/internal/services/filemanager"
	filemanagerv1 "github.com/IlianBuh/fmProto/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FileManager interface {
	GetFile(
		context.Context,
		string,
		filemanager.Sender,
	) error
	PostFile(
		ctx context.Context,
		recv filemanager.Receiver,
	) error
	DeleteFile(
		ctx context.Context,
		fileName string,
	) error
	PutFile(
		ctx context.Context,
		recv filemanager.Receiver,
	) error
}

type serverAPI struct {
	fm FileManager
	filemanagerv1.UnimplementedFileManagerServer
}

func Register(gRPC *grpc.Server, FM FileManager) {
	filemanagerv1.RegisterFileManagerServer(gRPC, &serverAPI{fm: FM})
}

func (s *serverAPI) GetFile(req *filemanagerv1.GetFileRequest, stream grpc.ServerStreamingServer[filemanagerv1.GetFileResponse]) error {

	err := s.fm.GetFile(
		context.Background(),
		req.GetFileName(),
		&wrappers.MyGetFileResponse{Stream: stream},
	)
	if err != nil {
		if errors.Is(err, filemanager.ErrBadRequest) {
			return status.Error(codes.NotFound, "bad request")
		}

		return status.Errorf(codes.Internal, "internal server error: %v", err)
	}

	return nil
}

// PostFile gets stream from the grpc client and receives data
//
// API error codes: DataLoss, Internal, InvalidArgument
func (s *serverAPI) PostFile(
	stream grpc.ClientStreamingServer[
		filemanagerv1.PostFileRequest,
		filemanagerv1.PostFileResponse,
	],
) error {

	defer func() {
		stream.SendAndClose(nil)
	}()
	err := s.fm.PostFile(
		context.Background(),
		&wrappers.MyPostFileProvider{Stream: stream},
	)
	if err != nil {
		switch {
		case errors.Is(err, filemanager.ErrReceiveFile):
			return status.Error(codes.DataLoss, "failed to get chunk")
		case errors.Is(err, filemanager.ErrInternal):
			return status.Error(codes.Internal, "internal error")
		case errors.Is(err, filemanager.ErrBadRequest):
			return status.Error(codes.InvalidArgument, "bad request")
		}

		return status.Error(codes.Internal, "failed to save file")
	}

	return nil
}

func (s *serverAPI) DeleteFile(
	ctx context.Context,
	req *filemanagerv1.DeleteFileRequest,
) (*filemanagerv1.DeleteFileResponse, error) {

	err := s.fm.DeleteFile(ctx, req.GetFileName())
	if err != nil {
		if errors.Is(err, filemanager.ErrBadRequest) {
			return nil, status.Error(codes.InvalidArgument, "file not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &filemanagerv1.DeleteFileResponse{}, nil
}

func (s *serverAPI) PutFile(
	stream grpc.ClientStreamingServer[filemanagerv1.PutFileRequest, filemanagerv1.PutFileResponse],
) error {
	err := s.fm.PutFile(
		context.Background(),
		&wrappers.MyPutFileProvider{Stream: stream},
	)
	if err != nil {
		switch {
		case errors.Is(err, filemanager.ErrReceiveFile):
			return status.Error(codes.DataLoss, "failed to get chunk")
		case errors.Is(err, filemanager.ErrInternal):
			return status.Error(codes.Internal, "internal error")
		case errors.Is(err, filemanager.ErrBadRequest):
			return status.Error(codes.InvalidArgument, "bad request")
		}

		return status.Error(codes.Internal, "failed to save file")
	}

	stream.SendAndClose(nil)
	return nil
}
