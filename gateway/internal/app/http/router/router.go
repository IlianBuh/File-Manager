package router

import (
	"github.com/go-chi/chi"
	grpclient "lab3/internal/clients/fm/grpc"
	"lab3/internal/handlers/http_handlers"
	"log/slog"
)

func NewRouter(log *slog.Logger, client *grpclient.Client) *chi.Mux {
	r := chi.NewRouter()

	bindMiddlewares(r, log)

	r.Route("/filemanager", func(c chi.Router) {
		c.Post("/", http_handlers.NewPost(log, client))
		c.Get("/", http_handlers.NewGet(log, client))
		c.Delete("/", http_handlers.NewDelete(log, client))
		c.Put("/", http_handlers.NewPut(log, client))
	})

	return r
}
