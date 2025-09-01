package main

import (
	"fmt"
	"os"
	"url-shortener/internal/config"

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
	// TODO: init logger: slog??? i'll use ZAP

	// TODO: init storage: sqlite... or postgres?

	// TODO: init router: chi, "chi render"

	// TODO: run server
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
