package main

import (
	"fmt"
	"net/http"
	"os"
	"url-shortener/internal/config"
	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/save"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/lib/logger/zp"
	"url-shortener/internal/storage/sqlite"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()
	log, _, err := setupLogger(cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	zap.ReplaceGlobals(log)
	zap.L().Info("starting url-shortener", zap.String("env", cfg.Env))
	zap.L().Debug("debug messages are enabled")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		zap.L().Error("failed to init storage", zp.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()
	// middleware
	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
		}))
		r.Post("/", save.New(log, storage))
		r.Delete("/{alias}", delete.New(log, storage))
	})
	router.Get("/{alias}", redirect.New(log, storage))

	//TODO: doesn't work at all
	//router.Delete("/url/{alias}", redirect.New(log, storage))

	log.Info("starting server", zap.String("address", cfg.Address))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	log.Error("server stopped")
}

func setupLogger(env string) (*zap.Logger, zap.AtomicLevel, error) {
	switch env {
	case envLocal:
		// console → stdout, debug
		return buildZap(
			zapcore.DebugLevel,
			newConsoleEncoder(true), // color
			zapcore.AddSync(os.Stdout),
		)

	case envDev:
		// json → dev.log, debug
		ws, err := fileSyncer("dev.log")
		if err != nil {
			return nil, zap.AtomicLevel{}, err
		}
		return buildZap(
			zapcore.DebugLevel,
			newJSONEncoder(),
			ws,
		)

	case envProd:
		// json → prod.log, info
		ws, err := fileSyncer("prod.log")
		if err != nil {
			return nil, zap.AtomicLevel{}, err
		}
		return buildZap(
			zapcore.InfoLevel,
			newJSONEncoder(),
			ws,
		)

	default:
		return nil, zap.AtomicLevel{}, fmt.Errorf("unknown env: %q", env)
	}
}

func newConsoleEncoder(color bool) zapcore.Encoder {
	enc := zap.NewDevelopmentEncoderConfig()
	enc.EncodeTime = zapcore.ISO8601TimeEncoder
	if color {
		enc.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	return zapcore.NewConsoleEncoder(enc)
}

func newJSONEncoder() zapcore.Encoder {
	enc := zap.NewProductionEncoderConfig()
	enc.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewJSONEncoder(enc)
}

func fileSyncer(path string) (zapcore.WriteSyncer, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", path, err)
	}
	return zapcore.AddSync(f), nil
}

func buildZap(level zapcore.Level, encoder zapcore.Encoder, sink zapcore.WriteSyncer) (*zap.Logger, zap.AtomicLevel, error) {
	atom := zap.NewAtomicLevelAt(level)

	core := zapcore.NewCore(encoder, sink, atom)
	lg := zap.New(core,
		zap.AddCaller(),                       // как у slog WithCaller
		zap.AddStacktrace(zapcore.ErrorLevel), // stacktrace c Error+
	)
	return lg, atom, nil
}
