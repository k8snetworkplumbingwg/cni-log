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

package state

import (
	"io"
	"sync"

	"github.com/k8snetworkplumbingwg/cni-log/pkg/level"
	"github.com/k8snetworkplumbingwg/cni-log/pkg/options"
	"github.com/k8snetworkplumbingwg/cni-log/pkg/prefixer"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var singleton state

// state is the struct for the l singleton. It allows us to set all logger attributes in a threadsafe manner for
// as long as we always access all of its attributes via its methods.
type state struct {
	loggerMutex        sync.RWMutex
	logger             *lumberjack.Logger
	logWriter          io.Writer
	logLevel           level.Level
	logToStderr        bool
	prefixer           prefixer.Prefixer
	structuredPrefixer prefixer.StructuredPrefixer
}

func Instance() *state {
	return &singleton
}

// setLogger sets l's logger.
func (s *state) SetLogger(logger *lumberjack.Logger) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logger = logger
}

// getLogger gets l's logger.
func (s *state) GetLogger() *lumberjack.Logger {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logger
}

// setLogWriter sets l's logWriter.
func (s *state) SetLogWriter(logWriter io.Writer) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logWriter = logWriter
}

// setLoggerAsLogWriter sets l's logWriter to a copy of its current logger.
func (s *state) SetLoggerAsLogWriter() {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	cp := *s.logger
	s.logWriter = &cp
}

// getLogWriter gets l's logWriter.
func (s *state) GetLogWriter() io.Writer {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logWriter
}

// setLogLevel sets l's logLevel.
func (s *state) SetLogLevel(logLevel level.Level) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logLevel = logLevel
}

// getLogLevel gets l's logLevel.
func (s *state) GetLogLevel() level.Level {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logLevel
}

// setLogToStderr sets l's logToStderr.
func (s *state) SetLogToStderr(logToStderr bool) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logToStderr = logToStderr
}

// getLogToStderr gets l's getLogToStderr.
func (s *state) GetLogToStderr() bool {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logToStderr
}

// setPrefixer sets l's prefixer.
func (s *state) SetPrefixer(p prefixer.Prefixer) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.prefixer = p
}

// getPrefixer gets l's prefixer.
func (s *state) GetPrefixer() prefixer.Prefixer {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.prefixer
}

// setStructuredPrefixer sets l's structuredPrefixer.
func (s *state) SetStructuredPrefixer(structuredPrefixer prefixer.StructuredPrefixer) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.structuredPrefixer = structuredPrefixer
}

// getStructuredPrefixer gets l's structuredPrefixer.
func (s *state) GetStructuredPrefixer() prefixer.StructuredPrefixer {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.structuredPrefixer
}

// isFileLoggingEnabled returns true if file logging is enabled.
func (s *state) IsFileLoggingEnabled() bool {
	return s.GetLogWriter() != nil
}

// setLogFile sets l's file logging. This method sets l.logger.Filename to the specified value and then sets l.logWriter
// to a copy of l.logger. If filename is the empty string, logWriter is set to nil.
func (s *state) SetLogFile(filename string) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	if filename == "" {
		s.logger.Filename = ""
		s.logWriter = nil
		return
	}
	s.logger.Filename = filename
	cp := *(s.logger)
	s.logWriter = &cp
}

// setLogOptions sets l's file logging options.
func (s *state) SetLogOptions(o *options.Options) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	// give some default value
	s.logger.MaxSize = 100
	s.logger.MaxAge = 5
	s.logger.MaxBackups = 5
	s.logger.Compress = true
	if o != nil {
		if o.MaxAge != nil {
			s.logger.MaxAge = *o.MaxAge
		}
		if o.MaxSize != nil {
			s.logger.MaxSize = *o.MaxSize
		}
		if o.MaxBackups != nil {
			s.logger.MaxBackups = *o.MaxBackups
		}
		if o.Compress != nil {
			s.logger.Compress = *o.Compress
		}
	}
}
