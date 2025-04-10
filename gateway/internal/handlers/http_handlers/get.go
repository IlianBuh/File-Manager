package http_handlers

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/fs"
	grpclient "lab3/internal/clients/fm/grpc"
	httperrors "lab3/internal/lib/http/errors"
	"lab3/internal/lib/logger/sl"
	"log/slog"
	"net/http"
)

func NewGet(log *slog.Logger, client *grpclient.Client) http.HandlerFunc {
	const method = "GET"
	log = log.With(slog.String("method", method))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var httpErrCode int
		log.Info("attempting to get file from grpc-server")

		filepath := r.URL.Query().Get("filepath")
		if !fs.ValidPath(filepath) {
			log.Warn("invalid file path", slog.String("filepath", filepath))
			httperrors.Error(w, http.StatusBadRequest)
			return
		}

		res, err := client.GetFile(context.Background(), filepath)
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				log.Warn("invalid file path", sl.Err(err))
				httpErrCode = http.StatusNotFound
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

		w.Header().Set("Content/type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath))
		w.WriteHeader(http.StatusOK)

		_, err = w.Write(res)
		if err != nil {
			log.Error("failed to write response", sl.Err(err))
			return
		}

		log.Info(
			"file successfully served",
			slog.String("filepath", filepath),
		)
	})
}
