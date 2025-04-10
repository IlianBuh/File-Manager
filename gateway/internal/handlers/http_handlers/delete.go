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
	"net/http"
)

func NewDelete(log *slog.Logger, client *grpclient.Client) http.HandlerFunc {
	const method = "DELETE"
	log = log.With(slog.String("method", method))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("attempting to delete file from grpc-server")
		var httpErrCode int
		filepath := r.URL.Query().Get("filepath")
		if !fs.ValidPath(filepath) {
			log.Warn("invalid filepath", slog.String("filepath", filepath))
			httperrors.Error(w, http.StatusBadRequest)
		}

		err := client.DeleteFile(context.Background(), filepath)
		if err != nil {
			switch status.Code(err) {
			case codes.InvalidArgument:
				log.Warn("bad request", sl.Err(err))
				httpErrCode = http.StatusBadRequest
			case codes.Internal:
				log.Error("internal error from grpc server is received", sl.Err(err))
				httpErrCode = http.StatusInternalServerError
			default:
				log.Error("unexpected error from gRPC server", sl.Err(err))
				httpErrCode = http.StatusInternalServerError
			}

			httperrors.Error(w, httpErrCode)
			return
		}

		log.Info("successfully deleted file from grpc-server")
		w.WriteHeader(http.StatusNoContent)
	})
}
