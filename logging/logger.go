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
// The environment variable `GNET_LOGGING_LEVEL` determines which zap logger level will be applied for logging.
// The environment variable `GNET_LOGGING_FILE` is set to a local file path when you want to print logs into local file.
// Alternatives of logging level (the variable of logging level ought to be integer):
/*
const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel
)
*/
package logging

import (
	"errors"
	"os"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	flushLogs           func() error
	defaultLogger       Logger
	defaultLoggingLevel zapcore.Level
)

func init() {
	lvl := os.Getenv("GNET_LOGGING_LEVEL")
	if len(lvl) > 0 {
		loggingLevel, err := strconv.ParseInt(lvl, 10, 8)
		if err != nil {
			panic("invalid GNET_LOGGING_LEVEL, " + err.Error())
		}
		defaultLoggingLevel = zapcore.Level(loggingLevel)
	}

	// Initializes the inside default logger of client.
	fileName := os.Getenv("GNET_LOGGING_FILE")
	if len(fileName) > 0 {
		var err error
		defaultLogger, flushLogs, err = CreateLoggerAsLocalFile(fileName, defaultLoggingLevel)
		if err != nil {
			panic("invalid GNET_LOGGING_FILE, " + err.Error())
		}
	} else {
		cfg := zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(defaultLoggingLevel)
		zapLogger, _ := cfg.Build()
		defaultLogger = zapLogger.Sugar()
	}
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// GetDefaultLogger returns the default logger.
func GetDefaultLogger() Logger {
	return defaultLogger
}

// LogLevel tells what the default logging level is.
func LogLevel() string {
	return defaultLoggingLevel.String()
}

// CreateLoggerAsLocalFile setups the logger by local file path.
func CreateLoggerAsLocalFile(localFilePath string, logLevel zapcore.Level) (logger Logger, flush func() error, err error) {
	if len(localFilePath) == 0 {
		return nil, nil, errors.New("invalid local logger path")
	}

	// lumberjack.Logger is already safe for concurrent use, so we don't need to lock it.
	lumberJackLogger := &lumberjack.Logger{
		Filename:   localFilePath,
		MaxSize:    100, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	}

	encoder := getEncoder()
	ws := zapcore.AddSync(lumberJackLogger)
	zapcore.Lock(ws)

	levelEnabler := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= logLevel
	})
	core := zapcore.NewCore(encoder, ws, levelEnabler)
	zapLogger := zap.New(core, zap.AddCaller())
	logger = zapLogger.Sugar()
	flush = zapLogger.Sync
	return
}

// Cleanup does something windup for logger, like closing, flushing, etc.
func Cleanup() {
	if flushLogs != nil {
		_ = flushLogs()
	}
}

// LogErr prints err if it's not nil.
func LogErr(err error) {
	if err != nil {
		defaultLogger.Errorf("error occurs during runtime, %v", err)
	}
}

// Debugf logs messages at DEBUG level.
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Infof logs messages at INFO level.
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warnf logs messages at WARN level.
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Errorf logs messages at ERROR level.
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatalf logs messages at FATAL level.
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
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
