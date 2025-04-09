package httpapp

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"lab3/internal/app/http/router"
	grpclient "lab3/internal/clients/fm/grpc"
	"lab3/internal/lib/logger/sl"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type App struct {
	log     *slog.Logger
	r       chi.Router
	httpSrv *http.Server
}

func New(
	log *slog.Logger,
	port string,
	addr string,
	idleTimout time.Duration,
	timeout time.Duration,
	client *grpclient.Client,
) *App {
	r := router.NewRouter(log, client)

	httpSrv := &http.Server{
		Addr:         getAddr(addr, port),
		Handler:      r,
		IdleTimeout:  idleTimout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	return &App{
		log:     log,
		r:       r,
		httpSrv: httpSrv,
	}
}

func getAddr(addr, port string) string {
	return net.JoinHostPort(addr, port)
}

// MustRun is Run wrapper.
// If Run ends with error panic occurs
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Run runs http application.
func (a *App) Run() error {
	const op = "httpapp.Run"
	log := a.log.With(slog.String("op", op))
	log.Info("starting http application")

	if err := a.httpSrv.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		log.Error("http server stopped with error", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Stop stops http application with graceful shutdown
func (a *App) Stop() {
	const op = "httpapp.Stop"
	log := a.log.With(slog.String("op", op))
	log.Info("stopping http application")

	ctx, cancel := context.WithTimeout(context.Background(), a.httpSrv.IdleTimeout)
	defer cancel()

	go func() {

		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			log.Error("context deadline exceeded", sl.Err(ctx.Err()))
			panic(ctx.Err())
		}

	}()

	if err := a.httpSrv.Shutdown(ctx); err != nil {
		log.Error("failed to stop http application", sl.Err(err))
	}

	log.Info("stopped http application")
}
