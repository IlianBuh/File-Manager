package main

import (
	"github.com/IlianBuh/filemanager-server/internal/app"
	"github.com/IlianBuh/filemanager-server/internal/config"
	"github.com/IlianBuh/filemanager-server/internal/lib/logger/slogpretty"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.New()

	log := setUpLogger(cfg)

	log.Info("logger initialized", slog.Any("cfg", cfg))

	application := app.New(log, cfg.GRPCObj.Port, cfg.RootPath, cfg.GRPCObj.Timeout)

	go application.GRPCApp.MustRun()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sign := <-stop
	log.Info("received signal", slog.Any("signal", sign))
	application.GRPCApp.Stop()
}

func setUpLogger(cfg *config.Config) *slog.Logger {
	var log *slog.Logger

	switch cfg.Env {
	case envLocal:
		log = setUpPrettyLogger()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setUpPrettyLogger() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	log := opts.NewPrettyHandler(os.Stdout)

	return slog.New(log)
}
