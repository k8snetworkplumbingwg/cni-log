// Copyright (c) 2018 Intel Corporation
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

package logging

import (
	"io"
	"sync"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// logging is the struct for the l singleton. It allows us to set all logger attributes in a threadsafe manner for
// as long as we always access all of its attributes via its methods.
type logging struct {
	loggerMutex        sync.RWMutex
	logger             *lumberjack.Logger
	logWriter          io.Writer
	logLevel           Level
	logToStderr        bool
	prefixer           Prefixer
	structuredPrefixer StructuredPrefixer
}

// setLogger sets l's logger.
func (l *logging) setLogger(logger *lumberjack.Logger) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	l.logger = logger
}

// getLogger gets l's logger.
func (l *logging) getLogger() *lumberjack.Logger {
	l.loggerMutex.RLock()
	defer l.loggerMutex.RUnlock()

	return l.logger
}

// setLogWriter sets l's logWriter.
func (l *logging) setLogWriter(logWriter io.Writer) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	l.logWriter = logWriter
}

// setLoggerAsLogWriter sets l's logWriter to a copy of its current logger.
func (l *logging) setLoggerAsLogWriter() {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	cp := *l.logger
	l.logWriter = &cp
}

// getLogWriter gets l's logWriter.
func (l *logging) getLogWriter() io.Writer {
	l.loggerMutex.RLock()
	defer l.loggerMutex.RUnlock()

	return l.logWriter
}

// setLogLevel sets l's logLevel.
func (l *logging) setLogLevel(logLevel Level) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	l.logLevel = logLevel
}

// getLogLevel gets l's logLevel.
func (l *logging) getLogLevel() Level {
	l.loggerMutex.RLock()
	defer l.loggerMutex.RUnlock()

	return l.logLevel
}

// setLogToStderr sets l's logToStderr.
func (l *logging) setLogToStderr(logToStderr bool) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	l.logToStderr = logToStderr
}

// getLogToStderr gets l's getLogToStderr.
func (l *logging) getLogToStderr() bool {
	l.loggerMutex.RLock()
	defer l.loggerMutex.RUnlock()

	return l.logToStderr
}

// setPrefixer sets l's prefixer.
func (l *logging) setPrefixer(prefixer Prefixer) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	l.prefixer = prefixer
}

// getPrefixer gets l's prefixer.
func (l *logging) getPrefixer() Prefixer {
	l.loggerMutex.RLock()
	defer l.loggerMutex.RUnlock()

	return l.prefixer
}

// setStructuredPrefixer sets l's structuredPrefixer.
func (l *logging) setStructuredPrefixer(structuredPrefixer StructuredPrefixer) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	l.structuredPrefixer = structuredPrefixer
}

// getStructuredPrefixer gets l's structuredPrefixer.
func (l *logging) getStructuredPrefixer() StructuredPrefixer {
	l.loggerMutex.RLock()
	defer l.loggerMutex.RUnlock()

	return l.structuredPrefixer
}

// isFileLoggingEnabled returns true if file logging is enabled.
func (l *logging) isFileLoggingEnabled() bool {
	return l.getLogWriter() != nil
}

// setLogFile sets l's file logging. This method sets l.logger.Filename to the specified value and then sets l.logWriter
// to a copy of l.logger. If filename is the empty string, logWriter is set to nil.
func (l *logging) setLogFile(filename string) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	if filename == "" {
		l.logger.Filename = ""
		l.logWriter = nil
		return
	}
	l.logger.Filename = filename
	cp := *(l.logger)
	l.logWriter = &cp
}

// setLogOpt8ions sets l's file logging options.
func (l *logging) setLogOptions(options *LogOptions) {
	l.loggerMutex.Lock()
	defer l.loggerMutex.Unlock()

	// give some default value
	l.logger.MaxSize = 100
	l.logger.MaxAge = 5
	l.logger.MaxBackups = 5
	l.logger.Compress = true
	if options != nil {
		if options.MaxAge != nil {
			l.logger.MaxAge = *options.MaxAge
		}
		if options.MaxSize != nil {
			l.logger.MaxSize = *options.MaxSize
		}
		if options.MaxBackups != nil {
			l.logger.MaxBackups = *options.MaxBackups
		}
		if options.Compress != nil {
			l.logger.Compress = *options.Compress
		}
	}
}
