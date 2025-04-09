package main

import (
	"lab3/internal/app"
	"lab3/internal/config"
	"lab3/internal/lib/logger/handler/slogpretty"
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

	log.Info("logger and config are initialized", slog.Any("config", cfg))

	application := app.New(
		log,
		cfg.FmPort,
		cfg.HTTPSrv.Port,
		cfg.HTTPSrv.Addr,
		cfg.HTTPSrv.IdleTimeout,
		cfg.HTTPSrv.Timeout,
		cfg.RetriesCount,
	)

	go application.HTTPApp.MustRun()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sign := <-stop
	log.Info("received signal", slog.String("signal", sign.String()))

	application.GRPCClient.Stop()
	application.HTTPApp.Stop()
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
