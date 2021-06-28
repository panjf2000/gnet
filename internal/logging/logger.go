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

// Package logging provides logging functionality for gnet server,
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
package logging

import (
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// DefaultLogger is the default logger inside the tbuspp client.
	DefaultLogger Logger
	zapLogger     *zap.Logger
	loggingLevel  zapcore.Level
)

// Init initializes the inside default logger of client.
func Init(logLevel zapcore.Level) {
	cfg := zap.NewDevelopmentConfig()
	loggingLevel = logLevel
	cfg.Level = zap.NewAtomicLevelAt(logLevel)
	zapLogger, _ = cfg.Build()
	DefaultLogger = zapLogger.Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// SetupLoggerWithPath setups the logger by local file path.
func SetupLoggerWithPath(localPath string, logLevel zapcore.Level) (err error) {
	if len(localPath) == 0 {
		return errors.New("invalid local logger path")
	}

	// lumberjack.Logger is already safe for concurrent use, so we don't need to lock it.
	w := &lumberjack.Logger{
		Filename:   localPath,
		MaxSize:    100, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	}

	loggingLevel = logLevel
	encoder := getEncoder()
	syncer := zapcore.AddSync(w)
	highPriority := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= logLevel
	})
	core := zapcore.NewCore(encoder, syncer, highPriority)
	zapLogger := zap.New(core, zap.AddCaller())
	DefaultLogger = zapLogger.Sugar()
	return nil
}

// SetupLogger setups the logger by the Logger interface.
func SetupLogger(logger Logger, logLevel zapcore.Level) {
	if logger == nil {
		return
	}
	loggingLevel = logLevel
	zapLogger = nil
	DefaultLogger = logger
}

// Cleanup does something windup for logger, like closing, flushing, etc.
func Cleanup() {
	if zapLogger != nil {
		_ = zapLogger.Sync()
	}
}

// LogErr prints err if it's not nil.
func LogErr(err error) {
	if err != nil {
		DefaultLogger.Errorf("error occurs during runtime, %v", err)
	}
}

// Level returns the logging level.
func Level() zapcore.Level {
	return loggingLevel
}

// Debugf logs messages at DEBUG level.
func Debugf(format string, args ...interface{}) {
	DefaultLogger.Debugf(format, args...)
}

// Infof logs messages at INFO level.
func Infof(format string, args ...interface{}) {
	DefaultLogger.Infof(format, args...)
}

// Warnf logs messages at WARN level.
func Warnf(format string, args ...interface{}) {
	DefaultLogger.Warnf(format, args...)
}

// Errorf logs messages at ERROR level.
func Errorf(format string, args ...interface{}) {
	DefaultLogger.Errorf(format, args...)
}

// Fatalf logs messages at FATAL level.
func Fatalf(format string, args ...interface{}) {
	DefaultLogger.Errorf(format, args...)
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
