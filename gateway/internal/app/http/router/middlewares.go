package router

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	middlewareLogger "lab3/internal/lib/logger/middleware"
	"log/slog"
	"net/http"
)

func bindMiddlewares(r *chi.Mux, log *slog.Logger) {

	r.Use(middleware.RequestID)
	r.Use(middlewareLogger.New(log))
	r.Use(cors)
	r.Use(middleware.Recoverer)

}

func cors(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST,GET,DELETE,PUT")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
