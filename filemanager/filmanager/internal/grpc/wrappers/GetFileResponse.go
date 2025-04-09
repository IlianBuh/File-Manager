package wrappers

import (
	filemanagerv1 "github.com/IlianBuh/fmProto/gen/go"
	"google.golang.org/grpc"
)

type gfres = filemanagerv1.GetFileResponse
type MyGetFileResponse struct {
	Stream grpc.ServerStreamingServer[gfres]
}

func (g *MyGetFileResponse) MySend(chunk []byte) error {
	return g.Stream.Send(&gfres{Chunk: chunk})
}
