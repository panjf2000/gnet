// Copyright (c) 2020 The Gnet Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Flusher is the callback function which flushes any buffered log entries to the underlying writer.
// It is usually called before the gnet process exits.
type Flusher = func() error

var (
	defaultLogger       Logger
	defaultLoggingLevel Level
	defaultFlusher      Flusher
)

// Level is the alias of zapcore.Level.
type Level = zapcore.Level

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = zapcore.DebugLevel
	// InfoLevel is the default logging priority.
	InfoLevel = zapcore.InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel = zapcore.WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel = zapcore.ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel = zapcore.DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel = zapcore.PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel = zapcore.FatalLevel
)

func init() {
	lvl := os.Getenv("GNET_LOGGING_LEVEL")
	if len(lvl) > 0 {
		loggingLevel, err := strconv.ParseInt(lvl, 10, 8)
		if err != nil {
			panic("invalid GNET_LOGGING_LEVEL, " + err.Error())
		}
		defaultLoggingLevel = Level(loggingLevel)
	}

	// Initializes the inside default logger of gnet.
	fileName := os.Getenv("GNET_LOGGING_FILE")
	if len(fileName) > 0 {
		var err error
		defaultLogger, defaultFlusher, err = CreateLoggerAsLocalFile(fileName, defaultLoggingLevel)
		if err != nil {
			panic("invalid GNET_LOGGING_FILE, " + err.Error())
		}
	} else {
		core := zapcore.NewCore(getDevEncoder(), zapcore.Lock(os.Stdout), defaultLoggingLevel)
		zapLogger := zap.New(core,
			zap.Development(),
			zap.AddCaller(),
			zap.AddStacktrace(ErrorLevel),
			zap.ErrorOutput(zapcore.Lock(os.Stderr)))
		defaultLogger = zapLogger.Sugar()
	}
}

type prefixEncoder struct {
	zapcore.Encoder

	prefix  string
	bufPool buffer.Pool
}

func (e *prefixEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := e.bufPool.Get()

	buf.AppendString(e.prefix)
	buf.AppendString(" ")

	logEntry, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(logEntry.Bytes())
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func getDevEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return &prefixEncoder{
		Encoder: zapcore.NewConsoleEncoder(encoderConfig),
		prefix:  "[gnet]",
		bufPool: buffer.NewPool(),
	}
}

func getProdEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return &prefixEncoder{
		Encoder: zapcore.NewConsoleEncoder(encoderConfig),
		prefix:  "[gnet]",
		bufPool: buffer.NewPool(),
	}
}

// GetDefaultLogger returns the default logger.
func GetDefaultLogger() Logger {
	return defaultLogger
}

// GetDefaultFlusher returns the default flusher.
func GetDefaultFlusher() Flusher {
	return defaultFlusher
}

var setupOnce sync.Once

// SetDefaultLoggerAndFlusher sets the default logger and its flusher.
//
// Note that this function should only be called once at the
// start of the program and not thereafter for the entire runtime,
// otherwise it will only keep the first setup.
func SetDefaultLoggerAndFlusher(logger Logger, flusher Flusher) {
	setupOnce.Do(func() {
		defaultLogger, defaultFlusher = logger, flusher
	})
}

// LogLevel tells what the default logging level is.
func LogLevel() string {
	return defaultLoggingLevel.String()
}

// CreateLoggerAsLocalFile setups the logger by local file path.
func CreateLoggerAsLocalFile(localFilePath string, logLevel Level) (logger Logger, flush func() error, err error) {
	if len(localFilePath) == 0 {
		return nil, nil, errors.New("invalid local logger path")
	}

	// lumberjack.Logger is already safe for concurrent use, so we don't need to lock it.
	lumberJackLogger := &lumberjack.Logger{
		Filename:   localFilePath,
		MaxSize:    100, // megabytes
		MaxBackups: 2,
		MaxAge:     15, // days
	}

	encoder := getProdEncoder()
	ws := zapcore.AddSync(lumberJackLogger)
	zapcore.Lock(ws)

	levelEnabler := zap.LevelEnablerFunc(func(level Level) bool {
		return level >= logLevel
	})
	core := zapcore.NewCore(encoder, ws, levelEnabler)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(ErrorLevel))
	logger = zapLogger.Sugar()
	flush = zapLogger.Sync
	return
}

// Cleanup does something windup for logger, like closing, flushing, etc.
func Cleanup() {
	if defaultFlusher != nil {
		_ = defaultFlusher()
	}
}

// Error prints err if it's not nil.
func Error(err error) {
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
	defaultLogger.Fatalf(format, args...)
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
