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
type Level int

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
	PanicLevel   Level = 1
	ErrorLevel   Level = 2
	WarningLevel Level = 3
	InfoLevel    Level = 4
	DebugLevel   Level = 5
	VerboseLevel Level = 6
	maximumLevel Level = VerboseLevel
)

const (
	defaultLogLevel        = InfoLevel
	defaultTimestampFormat = time.RFC3339

	logFileReqFailMsg  = "cni-log: filename is required\n"
	logFileFailMsg     = "cni-log: failed to set log file '%s'\n"
	setLevelFailMsg    = "cni-log: cannot set logging level to '%s'\n"
	symlinkEvalFailMsg = "cni-log: unable to evaluate symbolic links on path '%v'\n"
	emptyStringFailMsg = "cni-log: unable to resolve empty string"
)

var levelMap = map[string]Level{
	"panic":   PanicLevel,
	"error":   ErrorLevel,
	"warning": WarningLevel,
	"info":    InfoLevel,
	"debug":   DebugLevel,
	"verbose": VerboseLevel,
}

var logger *lumberjack.Logger
var logWriter io.Writer
var logLevel Level
var logToStderr bool
var prefixer Prefixer

// Prefix creator interface. Implement this interface if you wish to to create a custom prefix.
type Prefixer interface {
	// Produces the prefix string. CNI-Log will call this function
	// to request for the prefix when building the logging output and will pass in the appropriate
	// log level of your log message.
	CreatePrefix(Level) string
}

// Defines a default prefixer which will be used if a custom prefix is not provided
type defaultPrefixer struct {
	prefixFormat string
	timeFormat   string
}

// LogOptions defines the configuration of the lumberjack logger
type LogOptions struct {
	MaxAge     *int  `json:"maxAge,omitempty"`
	MaxSize    *int  `json:"maxSize,omitempty"`
	MaxBackups *int  `json:"maxBackups,omitempty"`
	Compress   *bool `json:"compress,omitempty"`
}

func init() {
	initLogger()
}

func initLogger() {
	logger = &lumberjack.Logger{}

	// Set default options.
	SetLogOptions(nil)
	SetLogStderr(true)
	SetLogFile("")
	SetLogLevel(defaultLogLevel)

	// Create the default prefixer
	SetDefaultPrefixer()
}

func (p *defaultPrefixer) CreatePrefix(loggingLevel Level) string {
	return fmt.Sprintf(p.prefixFormat, time.Now().Format(p.timeFormat), loggingLevel)
}

func SetPrefixer(p Prefixer) {
	prefixer = p
}

func SetDefaultPrefixer() {
	defaultPrefix := &defaultPrefixer{
		prefixFormat: "%s [%s] ",
		timeFormat:   defaultTimestampFormat,
	}
	SetPrefixer(defaultPrefix)
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

	if fp == "" {
		fmt.Fprint(os.Stderr, logFileReqFailMsg)
		return
	}

	if !isLogFileWritable(fp) {
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
func SetLogLevel(level Level) {
	if validateLogLevel(level) {
		logLevel = level
	} else {
		fmt.Fprintf(os.Stderr, setLevelFailMsg, level)
	}
}

func StringToLevel(level string) Level {
	if l, found := levelMap[strings.ToLower(level)]; found {
		return l
	}

	fmt.Fprintf(os.Stderr, setLevelFailMsg, level)
	return -1
}

// SetLogStderr sets flag for logging stderr output
func SetLogStderr(enable bool) {
	logToStderr = enable
}

func (l Level) String() string {
	switch l {
	case PanicLevel:
		return "panic"
	case VerboseLevel:
		return "verbose"
	case WarningLevel:
		return "warning"
	case InfoLevel:
		return "info"
	case ErrorLevel:
		return "error"
	case DebugLevel:
		return "debug"
	default:
		return "unknown"
	}
}

// SetOutput set custom output WARNING subsequent call to SetLogFile or SetLogOptions invalidates this setting
func SetOutput(out io.Writer) {
	logWriter = out
}

// Panicf prints logging plus stack trace. This should be used only for unrecoverable error
func Panicf(format string, a ...interface{}) {
	printf(PanicLevel, format, a...)
	printf(PanicLevel, "========= Stack trace output ========")
	printf(PanicLevel, "%+v", Errorf("CNI Panic"))
	printf(PanicLevel, "========= Stack trace output end ========")
}

// Errorf prints logging if logging level >= error
func Errorf(format string, a ...interface{}) error {
	printf(ErrorLevel, format, a...)
	return fmt.Errorf(format, a...)
}

// Warningf prints logging if logging level >= warning
func Warningf(format string, a ...interface{}) {
	printf(WarningLevel, format, a...)
}

// Infof prints logging if logging level >= info
func Infof(format string, a ...interface{}) {
	printf(InfoLevel, format, a...)
}

// Debugf prints logging if logging level >= debug
func Debugf(format string, a ...interface{}) {
	printf(DebugLevel, format, a...)
}

// Verbosef prints logging if logging level >= verbose
func Verbosef(format string, a ...interface{}) {
	printf(VerboseLevel, format, a...)
}

func doWritef(writer io.Writer, level Level, format string, a ...interface{}) {
	fmt.Fprint(writer, prefixer.CreatePrefix(level))
	fmt.Fprintf(writer, format, a...)
	fmt.Fprintf(writer, "\n")
}

func printf(level Level, format string, a ...interface{}) {
	if logger.Filename == "" {
		fmt.Fprint(os.Stderr, logFileReqFailMsg)
		return
	}

	if level > logLevel {
		return
	}

	if logToStderr {
		doWritef(os.Stderr, level, format, a...)
	}

	if logWriter != nil {
		doWritef(logWriter, level, format, a...)
	}
}

// isLogFileWritable checks if the path can be written to. If the file does not exist yet, the entire path including
// the file will be created.
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

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return false
	}
	f.Close()

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

func validateLogLevel(level Level) bool {
	return level > 0 && level <= maximumLevel
}
