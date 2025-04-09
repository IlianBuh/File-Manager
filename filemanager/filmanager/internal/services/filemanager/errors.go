package filemanager

import "errors"

var (
	ErrReceiveFile = errors.New("failed to download file chunk")
	ErrBadRequest  = errors.New("bad request")
	ErrInternal    = errors.New("internal error occurred")
)
