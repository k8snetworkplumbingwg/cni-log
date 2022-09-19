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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// Level type
type Level uint8

/*
Common use of different level:

"panic":   Code crash
"error":   Unusual event occurred (invalid input or system issue), so exiting code prematurely
"warning": Unusual event occurred (invalid input or system issue), but continuing
"info":    Basic information, indication of major code paths
"debug":   Additional information, indication of minor code branches
"verbose": Output of larger variables in code and debug of low level functions
*/

const (
	panicLevel Level = iota
	errorLevel
	warningLevel
	infoLevel
	debugLevel
	verboseLevel
	unknownLevel
)

const (
	defaultLogFile         = "/var/log/cni-log.log"
	defaultLogLevel        = infoLevel
	defaultTimestampFormat = time.RFC3339

	logFileFailMsg     = "cni-log: failed to set log file '%s'\n"
	setLevelFailMsg    = "cni-log: cannot set logging level to '%s'\n"
	symlinkEvalFailMsg = "cni-log: unable to evaluate symbolic links on path '%v'\n"
	emptyStringFailMsg = "cni-log: unable to resolve empty string"
)

var levelMap = map[string]Level{
	"panic":   panicLevel,
	"error":   errorLevel,
	"warning": warningLevel,
	"info":    infoLevel,
	"debug":   debugLevel,
	"verbose": verboseLevel,
}

var logger *lumberjack.Logger
var logWriter io.Writer
var logLevel Level
var logToStderr bool

// LogOptions defines the configuration of the lumberjack logger
type LogOptions struct {
	MaxAge     *int  `json:"maxAge,omitempty"`
	MaxSize    *int  `json:"maxSize,omitempty"`
	MaxBackups *int  `json:"maxBackups,omitempty"`
	Compress   *bool `json:"compress,omitempty"`
}

func init() {
	logToStderr = true
	logLevel = defaultLogLevel
	logger = &lumberjack.Logger{}

	// Setting default LogFile
	SetLogFile(defaultLogFile)
}

// Set the logging options (LogOptions)
func SetLogOptions(options *LogOptions) {
	// give some default value
	logger.MaxSize = 100
	logger.MaxAge = 5
	logger.MaxBackups = 5
	logger.Compress = true
	if options != nil {
		if options.MaxAge != nil {
			logger.MaxAge = *options.MaxAge
		}
		if options.MaxSize != nil {
			logger.MaxSize = *options.MaxSize
		}
		if options.MaxBackups != nil {
			logger.MaxBackups = *options.MaxBackups
		}
		if options.Compress != nil {
			logger.Compress = *options.Compress
		}
	}
	logWriter = logger
}

// SetLogFile sets logging file
func SetLogFile(filename string) {
	fp := resolvePath(filename)
	if fp == "" || !isLogFileWritable(fp) {
		fmt.Fprintf(os.Stderr, logFileFailMsg, filename)
		return
	}

	logger.Filename = filename
	logWriter = logger
}

// GetLogLevel gets current logging level
func GetLogLevel() Level {
	return logLevel
}

// SetLogLevel sets logging level
func SetLogLevel(level string) {
	l := convertLevelString(level)
	if l < unknownLevel {
		logLevel = l
	}
}

// SetLogStderr sets flag for logging stderr output
func SetLogStderr(enable bool) {
	logToStderr = enable
}

func (l Level) String() string {
	switch l {
	case panicLevel:
		return "panic"
	case verboseLevel:
		return "verbose"
	case warningLevel:
		return "warning"
	case infoLevel:
		return "info"
	case errorLevel:
		return "error"
	case debugLevel:
		return "debug"
	default:
		return "unknown"
	}
}

// Panicf prints logging plus stack trace. This should be used only for unrecoverable error
func Panicf(format string, a ...interface{}) {
	printf(panicLevel, format, a...)
	printf(panicLevel, "========= Stack trace output ========")
	printf(panicLevel, "%+v", Errorf("CNI Panic"))
	printf(panicLevel, "========= Stack trace output end ========")
}

// Errorf prints logging if logging level >= error
func Errorf(format string, a ...interface{}) error {
	printf(errorLevel, format, a...)
	return fmt.Errorf(format, a...)
}

// Warningf prints logging if logging level >= warning
func Warningf(format string, a ...interface{}) {
	printf(warningLevel, format, a...)
}

// Infof prints logging if logging level >= info
func Infof(format string, a ...interface{}) {
	printf(infoLevel, format, a...)
}

// Debugf prints logging if logging level >= debug
func Debugf(format string, a ...interface{}) {
	printf(debugLevel, format, a...)
}

// Verbosef prints logging if logging level >= verbose
func Verbosef(format string, a ...interface{}) {
	printf(verboseLevel, format, a...)
}

func convertLevelString(level string) Level {
	if l, found := levelMap[strings.ToLower(level)]; found {
		return l
	}

	fmt.Fprintf(os.Stderr, setLevelFailMsg, level)
	return unknownLevel
}

func doWrite(writer io.Writer, level Level, format string, a ...interface{}) {
	header := "%s [%s] "
	t := time.Now()

	fmt.Fprintf(writer, header, t.Format(defaultTimestampFormat), level)
	fmt.Fprintf(writer, format, a...)
	fmt.Fprintf(writer, "\n")
}

func printf(level Level, format string, a ...interface{}) {
	if level > logLevel {
		return
	}

	if logToStderr {
		doWrite(os.Stderr, level, format, a...)
	}

	if logWriter != nil {
		doWrite(logWriter, level, format, a...)
	}
}

func isLogFileWritable(filename string) bool {
	logFileDirs := filepath.Dir(filename)

	// Check if parent directories of log file exists
	// If not exist, try to create the parent directories.
	// If exists, check that a log file can be created in that directory
	if _, err := os.Stat(logFileDirs); os.IsNotExist(err) {
		if err = os.MkdirAll(logFileDirs, 0755); err != nil {
			// failed to create parent dirs. Assuming no write permissions
			return false
		}
	}

	if _, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
		return false
	}

	return true
}

func isSymLink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		return true
	}

	return false
}

func resolvePath(path string) string {
	if path == "" {
		fmt.Fprintln(os.Stderr, emptyStringFailMsg)
		return ""
	}

	if isSymLink(path) {
		fmt.Fprintf(os.Stderr, symlinkEvalFailMsg, path)
		return ""
	}

	return filepath.Clean(path)
}
