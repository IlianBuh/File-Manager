package app

import (
	grpcapp "github.com/IlianBuh/filemanager-server/internal/app/grpc"
	"github.com/IlianBuh/filemanager-server/internal/services/filemanager"
	"log/slog"
	"time"
)

type App struct {
	GRPCApp *grpcapp.App
}

func New(
	log *slog.Logger,
	port string,
	rootPath string,
	timeout time.Duration,

) *App {

	fm := filemanager.New(log, rootPath, timeout)

	grpcapp := grpcapp.New(log, port, fm)
	return &App{
		GRPCApp: grpcapp,
	}
}
