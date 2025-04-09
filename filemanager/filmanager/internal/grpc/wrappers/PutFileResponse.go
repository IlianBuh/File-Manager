package wrappers

import (
	"github.com/IlianBuh/filemanager-server/internal/services/filemanager"
	filemanagerv1 "github.com/IlianBuh/fmProto/gen/go"
	"google.golang.org/grpc"
)

type ptfreq = filemanagerv1.PutFileRequest
type ptfres = filemanagerv1.PutFileResponse

type MyPutFileProvider struct {
	Stream grpc.ClientStreamingServer[ptfreq, ptfres]
}

func (g *MyPutFileProvider) MyReceive() (filemanager.FileProvider, error) {
	return g.Stream.Recv()
}

func (g *MyPutFileProvider) MySend(data []byte) error {
	return g.Stream.SendAndClose(
		&ptfres{
			Status: filemanagerv1.ResponseStatus(data[0]),
		},
	)
}
