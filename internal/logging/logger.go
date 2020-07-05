// Copyright (c) 2020 Andy Pan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// package logging provides logging functionality for gnet server,
// it sets up a default logger (powered by go.uber.org/zap)
// which is about to be used by gnet server, it also allows users
// to replace the default logger with their customized logger by just
// implementing the `Logger` interface and assign it to the functional option `Options.Logger`,
// pass it to `gnet.Serve` method.
//
// There are two logging modes in zap, instantiated by either NewProduction() or NewDevelopment(),
// the former builds a sensible production Logger that writes InfoLevel and above logs to standard error as JSON,
// it's a shortcut for NewProductionConfig().Build(...Option); the latter builds a development Logger
// that writes DebugLevel and above logs to standard error in a human-friendly format,
// it's a shortcut for NewDevelopmentConfig().Build(...Option).
//
// The environment variable `GNET_LOGGING_MODE` determines which zap logger type will be created for logging,
// "prod" (case-insensitive) means production logger while other values except "prod" including "dev" (case-insensitive)
// represent development logger.

package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
)

var (
	// DefaultLogger is the default logger inside the gnet server.
	DefaultLogger Logger
	zapLogger     *zap.Logger
)

func init() {
	switch strings.ToLower(os.Getenv("GNET_LOGGING_MODE")) {
	case "prod":
		zapLogger, _ = zap.NewProduction()
	default:
		// Other values except "Prod" create the development logger for gnet server.
		zapLogger, _ = zap.NewDevelopment()
	}
	DefaultLogger = zapLogger.Sugar()
}

// Cleanup does something windup for logger, like closing, flushing, etc.
func Cleanup() {
	_ = zapLogger.Sync()
}

// Logger is used for logging formatted messages.
type Logger interface {
	// Debugf logs messages at DEBUG level.
	Debugf(format string, args ...interface{})
	// Infof logs messages at INFO level.
	Infof(format string, args ...interface{})
	// Warnf logs messages at WARN level.
	Warnf(format string, args ...interface{})
	// Errorf logs messages at ERROR level.
	Errorf(format string, args ...interface{})
	// Fatalf logs messages at FATAL level.
	Fatalf(format string, args ...interface{})
}
