package grpcapp

import (
	"fmt"
	grpcfm "github.com/IlianBuh/filemanager-server/internal/grpc"
	"github.com/IlianBuh/filemanager-server/internal/lib/logger/sl"
	"github.com/IlianBuh/filemanager-server/internal/services/filemanager"
	"google.golang.org/grpc"
	"log/slog"
	"net"
)

type App struct {
	log     *slog.Logger
	port    string
	gRPCSrv *grpc.Server
}

func New(
	log *slog.Logger,
	port string,
	fm *filemanager.FileManager,
) *App {
	grpcsrv := grpc.NewServer()
	grpcfm.Register(grpcsrv, fm)

	return &App{
		log:     log,
		port:    port,
		gRPCSrv: grpcsrv,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"
	log := a.log.With(slog.String("op", op))
	log.Info("starting grpc application")

	lis, err := net.Listen("tcp", ":"+a.port)
	if err != nil {
		log.Error("failed to listen socket", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("application started")
	if err := a.gRPCSrv.Serve(lis); err != nil {
		log.Error("failed to serve", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.Info("stopping grpc application", slog.String("op", op))

	a.gRPCSrv.GracefulStop()
}
