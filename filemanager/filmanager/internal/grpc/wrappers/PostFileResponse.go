package wrappers

import (
	"github.com/IlianBuh/filemanager-server/internal/services/filemanager"
	filemanagerv1 "github.com/IlianBuh/fmProto/gen/go"
	"google.golang.org/grpc"
)

type pfreq = filemanagerv1.PostFileRequest
type pfres = filemanagerv1.PostFileResponse

type MyPostFileProvider struct {
	Stream grpc.ClientStreamingServer[pfreq, pfres]
}

func (g *MyPostFileProvider) MyReceive() (filemanager.FileProvider, error) {
	return g.Stream.Recv()
}

func (g *MyPostFileProvider) MySend(data []byte) error {
	return g.Stream.SendAndClose(
		&pfres{
			Status: filemanagerv1.ResponseStatus(data[0]),
		},
	)
}
