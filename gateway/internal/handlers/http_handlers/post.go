package http_handlers

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/fs"
	grpclient "lab3/internal/clients/fm/grpc"
	httperrors "lab3/internal/lib/http/errors"
	"lab3/internal/lib/logger/sl"
	"log/slog"
	"mime/multipart"
	"net/http"
)

type MyHeader struct {
	header *multipart.FileHeader
}

func (m *MyHeader) Name() string {
	return m.header.Filename
}
func (m *MyHeader) Size() int64 {
	return m.header.Size
}

func NewPost(log *slog.Logger, client *grpclient.Client) http.HandlerFunc {
	const method = "POST"
	log = log.With(slog.String("method", method))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("attempting to post file on the grpc-server")
		var httpErrCode int

		filepath := r.FormValue("filepath")
		if !fs.ValidPath(filepath) {
			log.Warn("invalid filepath", slog.String("filepath", filepath))
			httperrors.Error(w, http.StatusBadRequest)
			return
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			log.Error("failed to get file from form", sl.Err(err))
			httperrors.Error(w, http.StatusBadRequest)
			return
		}
		defer file.Close()

		err = client.PostFile(context.Background(), file, &MyHeader{fileHeader}, filepath)
		if err != nil {
			switch status.Code(err) {
			case codes.InvalidArgument:
				log.Warn("bad request", sl.Err(err))
				httpErrCode = http.StatusBadRequest
			case codes.DataLoss:
				log.Error("data was loss", sl.Err(err))
				httpErrCode = http.StatusInternalServerError
			case codes.Internal:
				log.Error("internal error from grpc server", sl.Err(err))
				httpErrCode = http.StatusInternalServerError
			default:
				log.Error("unexpected error from grpc server", sl.Err(err))
				httpErrCode = http.StatusInternalServerError
			}

			httperrors.Error(w, httpErrCode)
			return
		}

		w.WriteHeader(http.StatusCreated)
		log.Info("successfully got file from grpc-server")
	})
}
