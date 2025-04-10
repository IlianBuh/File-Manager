package app

import (
	httpapp "lab3/internal/app/http"
	grpclient "lab3/internal/clients/fm/grpc"
	"log/slog"
	"time"
)

type App struct {
	HTTPApp    *httpapp.App
	GRPCClient *grpclient.Client
}

func New(
	log *slog.Logger,
	fmPort string,
	port string,
	addr string,
	idleTimout time.Duration,
	timeout time.Duration,
	retriesCount int,
) *App {

	client, err := grpclient.New(
		log,
		fmPort,
		timeout,
		retriesCount,
	)
	if err != nil {
		panic(err)
	}

	application := httpapp.New(log, port, addr, idleTimout, timeout, client)
	return &App{
		HTTPApp:    application,
		GRPCClient: client,
	}
}
