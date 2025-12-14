package utils

import (
	"context"
	"fmt"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

// Define a custom type for our context key
type contextKey string

// loggerKey is the key used to store the logrus.Entry in the context.
const loggerKey = contextKey("logger")

func init() {
	logrus.Info("Configuring global logger...")

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})
}

// WithLogger returns a new context with the provided logger.
func WithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger retrieves the request-specific logger from the context.
// If no logger is found, it returns the standard logrus logger.
func GetLogger(ctx context.Context) *logrus.Entry {
	// See if a logger is in the context
	logger, ok := ctx.Value(loggerKey).(*logrus.Entry)
	if !ok {
		// No logger found, return a default
		return logrus.NewEntry(logrus.StandardLogger())
	}
	return logger
}
