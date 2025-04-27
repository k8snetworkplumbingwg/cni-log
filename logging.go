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
	"runtime/debug"
	"strings"
	"sync"
	"time"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultLogLevel        = InfoLevel
	defaultTimestampFormat = time.RFC3339Nano

	logFileReqFailMsg              = "cni-log: filename is required when logging to stderr is off - will not log anything\n"
	logFileFailMsg                 = "cni-log: failed to set log file '%s'\n"
	setLevelFailMsg                = "cni-log: cannot set logging level to '%s'\n"
	symlinkEvalFailMsg             = "cni-log: unable to evaluate symbolic links on path '%v'"
	emptyStringFailMsg             = "cni-log: unable to resolve empty string"
	structuredLoggingOddArguments  = "must provide an even number of arguments for structured logging"
	structuredPrefixerOddArguments = "prefixer must return an even number of arguments for structured logging"
)

var loggingState state

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
	loggingState.setLogger(&lumberjack.Logger{})

	// Set default options.
	SetLogOptions(nil)
	SetLogStderr(true)
	SetLogFile("")
	SetLogLevel(defaultLogLevel)

	// Create the default prefixer
	SetDefaultPrefixer()
	SetDefaultStructuredPrefixer()
}

// SetPrefixer allows overwriting the Prefixer with a custom one.
func SetPrefixer(p Prefixer) {
	loggingState.setPrefixer(p)
}

// SetStructuredPrefixer allows overwriting the StructuredPrefixer with a custom one.
func SetStructuredPrefixer(p StructuredPrefixer) {
	loggingState.setStructuredPrefixer(p)
}

// SetDefaultPrefixer sets the default Prefixer.
func SetDefaultPrefixer() {
	defaultPrefix := &defaultPrefixer{
		prefixFormat: "%s [%s] ",
		timeFormat:   defaultTimestampFormat,
	}
	SetPrefixer(defaultPrefix)
}

// SetDefaultStructuredPrefixer sets the default StructuredPrefixer.
func SetDefaultStructuredPrefixer() {
	defaultStructuredPrefix := &defaultPrefixer{
		timeFormat: defaultTimestampFormat,
	}
	SetStructuredPrefixer(defaultStructuredPrefix)
}

// Set the logging options (LogOptions)
func SetLogOptions(options *LogOptions) {
	loggingState.setLogOptions(options)
	// Update the logWriter if necessary.
	if loggingState.isFileLoggingEnabled() {
		loggingState.setLoggerAsLogWriter()
	}
}

// SetLogFile sets logging file.
func SetLogFile(filename string) {
	// Allow logging to stderr only. Print an error a single time when this is set to the empty string but stderr
	// logging is off.
	if filename == "" {
		if !loggingState.getLogToStderr() {
			fmt.Fprint(os.Stderr, logFileReqFailMsg)
		}
		loggingState.setLogFile("")
		return
	}

	fp, err := resolvePath(filename)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	if !isLogFileWritable(fp) {
		fmt.Fprintf(os.Stderr, logFileFailMsg, filename)
		return
	}

	loggingState.setLogFile(filename)
}

// GetLogLevel gets current logging level
func GetLogLevel() Level {
	return loggingState.getLogLevel()
}

// SetLogLevel sets logging level
func SetLogLevel(level Level) {
	if level.IsValid() {
		loggingState.setLogLevel(level)
	} else {
		fmt.Fprintf(os.Stderr, setLevelFailMsg, level)
	}
}

// SetLogStderr sets flag for logging stderr output
func SetLogStderr(enable bool) {
	if !enable && !loggingState.isFileLoggingEnabled() {
		fmt.Fprint(os.Stderr, logFileReqFailMsg)
	}
	loggingState.setLogToStderr(enable)
}

// SetOutput set custom output WARNING subsequent call to SetLogFile or SetLogOptions invalidates this setting
func SetOutput(out io.Writer) {
	loggingState.setLogWriter(out)
}

// Panicf prints logging plus stack trace. This should be used only for unrecoverable error
func Panicf(format string, a ...interface{}) {
	printf(PanicLevel, format, a...)
	printf(PanicLevel, "========= Stack trace output ========")
	printf(PanicLevel, "%+v", string(debug.Stack()))
	printf(PanicLevel, "========= Stack trace output end ========")
}

// PanicStructured provides structured logging for log level >= panic.
func PanicStructured(msg string, args ...interface{}) {
	stackTrace := string(debug.Stack())
	args = append(args, "stacktrace", stackTrace)
	m := structuredMessage(PanicLevel, msg, args...)
	printWithPrefixf(PanicLevel, false, m)
}

// Errorf prints logging if logging level >= error
func Errorf(format string, a ...interface{}) error {
	printf(ErrorLevel, format, a...)
	return fmt.Errorf(format, a...)
}

// ErrorStructured provides structured logging for log level >= error.
func ErrorStructured(msg string, args ...interface{}) error {
	m := structuredMessage(ErrorLevel, msg, args...)
	printWithPrefixf(ErrorLevel, false, m)
	return fmt.Errorf("%s", m)
}

// Warningf prints logging if logging level >= warning
func Warningf(format string, a ...interface{}) {
	printf(WarningLevel, format, a...)
}

// WarningStructured provides structured logging for log level >= warning.
func WarningStructured(msg string, args ...interface{}) {
	m := structuredMessage(WarningLevel, msg, args...)
	printWithPrefixf(WarningLevel, false, m)
}

// Infof prints logging if logging level >= info
func Infof(format string, a ...interface{}) {
	printf(InfoLevel, format, a...)
}

// InfoStructured provides structured logging for log level >= info.
func InfoStructured(msg string, args ...interface{}) {
	m := structuredMessage(InfoLevel, msg, args...)
	printWithPrefixf(InfoLevel, false, m)
}

// Debugf prints logging if logging level >= debug
func Debugf(format string, a ...interface{}) {
	printf(DebugLevel, format, a...)
}

// DebugStructured provides structured logging for log level >= debug.
func DebugStructured(msg string, args ...interface{}) {
	m := structuredMessage(DebugLevel, msg, args...)
	printWithPrefixf(DebugLevel, false, m)
}

// structuredMessage takes msg and an even list of args and returns a structured message.
func structuredMessage(loggingLevel Level, msg string, args ...interface{}) string {
	prefixArgs := loggingState.getStructuredPrefixer().CreateStructuredPrefix(loggingLevel, msg)
	if len(prefixArgs)%2 != 0 {
		panic(fmt.Sprintf("msg=%q logging_failure=%q", msg, structuredPrefixerOddArguments))
	}

	var output []string
	for i := 0; i < len(prefixArgs)-1; i += 2 {
		output = append(output, fmt.Sprintf("%s=%q", argToString(prefixArgs[i]), argToString(prefixArgs[i+1])))
	}

	if len(args)%2 != 0 {
		output = append(output, fmt.Sprintf("logging_failure=%q", structuredLoggingOddArguments))
		panic(strings.Join(output, " "))
	}

	for i := 0; i < len(args)-1; i += 2 {
		output = append(output, fmt.Sprintf("%s=%q", argToString(args[i]), argToString(args[i+1])))
	}

	return strings.Join(output, " ")
}

// argToString returns the string representation of the provided interface{}.
func argToString(arg interface{}) string {
	return fmt.Sprintf("%+v", arg)
}

// doWritef takes care of the low level writing to the output io.Writer.
func doWritef(writer io.Writer, format string, a ...interface{}) {
	fmt.Fprintf(writer, format, a...)
	fmt.Fprintf(writer, "\n")
}

// printf prints log messages if they match the configured log level. A configured prefix is prepended to messages.
func printf(level Level, format string, a ...interface{}) {
	printWithPrefixf(level, true, format, a...)
}

// printWithPrefixf prints log messages if they match the configured log level. Messages are optionally prepended by a
// configured prefix.
func printWithPrefixf(level Level, printPrefix bool, format string, a ...interface{}) {
	if level > loggingState.getLogLevel() {
		return
	}

	if !loggingState.isFileLoggingEnabled() && !loggingState.getLogToStderr() {
		return
	}

	if printPrefix {
		format = loggingState.getPrefixer().CreatePrefix(level) + format
	}

	if loggingState.getLogToStderr() {
		doWritef(os.Stderr, format, a...)
	}

	if loggingState.isFileLoggingEnabled() {
		doWritef(loggingState.getLogWriter(), format, a...)
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

// resolvePath will try to resolve the provided path. If path is empty or is a symlink, return an error.
func resolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf(emptyStringFailMsg)
	}

	if isSymLink(path) {
		return "", fmt.Errorf(symlinkEvalFailMsg, path)
	}

	return filepath.Clean(path), nil
}

// state is the struct for the loggingState singleton. It allows us to set all logger attributes in a threadsafe manner
// for as long as we always access all of its attributes via its methods.
type state struct {
	loggerMutex        sync.RWMutex
	logger             *lumberjack.Logger
	logWriter          io.Writer
	logLevel           Level
	logToStderr        bool
	prefixer           Prefixer
	structuredPrefixer StructuredPrefixer
}

// setLogger sets the logger.
func (s *state) setLogger(logger *lumberjack.Logger) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logger = logger
}

// getLogger gets the logger.
func (s *state) getLogger() *lumberjack.Logger {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logger
}

// setLogWriter sets the logWriter.
func (s *state) setLogWriter(logWriter io.Writer) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logWriter = logWriter
}

// setLoggerAsLogWriter sets the logWriter to a copy of its current logger.
func (s *state) setLoggerAsLogWriter() {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	cp := *s.logger
	s.logWriter = &cp
}

// getLogWriter gets the logWriter.
func (s *state) getLogWriter() io.Writer {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logWriter
}

// setLogLevel sets the logLevel.
func (s *state) setLogLevel(logLevel Level) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logLevel = logLevel
}

// getLogLevel gets the logLevel.
func (s *state) getLogLevel() Level {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logLevel
}

// setLogToStderr sets logToStderr.
func (s *state) setLogToStderr(logToStderr bool) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.logToStderr = logToStderr
}

// getLogToStderr gets getLogToStderr.
func (s *state) getLogToStderr() bool {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.logToStderr
}

// setPrefixer sets the prefixer.
func (s *state) setPrefixer(prefixer Prefixer) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.prefixer = prefixer
}

// getPrefixer gets the prefixer.
func (s *state) getPrefixer() Prefixer {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.prefixer
}

// setStructuredPrefixer sets the structuredPrefixer.
func (s *state) setStructuredPrefixer(structuredPrefixer StructuredPrefixer) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	s.structuredPrefixer = structuredPrefixer
}

// getStructuredPrefixer gets the structuredPrefixer.
func (s *state) getStructuredPrefixer() StructuredPrefixer {
	s.loggerMutex.RLock()
	defer s.loggerMutex.RUnlock()

	return s.structuredPrefixer
}

// isFileLoggingEnabled returns true if the logWriter is not nil.
func (s *state) isFileLoggingEnabled() bool {
	return s.getLogWriter() != nil
}

// setLogFile sets the log file. This method sets s.logger.Filename to the specified value and then sets s.logWriter
// to a copy of s.logger. If filename is the empty string, logWriter is set to nil.
func (s *state) setLogFile(filename string) {
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

// setLogOptions sets the log options.
func (s *state) setLogOptions(options *LogOptions) {
	s.loggerMutex.Lock()
	defer s.loggerMutex.Unlock()

	// give some default value
	s.logger.MaxSize = 100
	s.logger.MaxAge = 5
	s.logger.MaxBackups = 5
	s.logger.Compress = true
	if options != nil {
		if options.MaxAge != nil {
			s.logger.MaxAge = *options.MaxAge
		}
		if options.MaxSize != nil {
			s.logger.MaxSize = *options.MaxSize
		}
		if options.MaxBackups != nil {
			s.logger.MaxBackups = *options.MaxBackups
		}
		if options.Compress != nil {
			s.logger.Compress = *options.Compress
		}
	}
}
