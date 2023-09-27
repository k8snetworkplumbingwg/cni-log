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
	"time"

	"github.com/k8snetworkplumbingwg/cni-log/pkg/level"
	"github.com/k8snetworkplumbingwg/cni-log/pkg/options"
	"github.com/k8snetworkplumbingwg/cni-log/pkg/prefixer"
	"github.com/k8snetworkplumbingwg/cni-log/pkg/state"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultLogLevel        = level.Info
	defaultTimestampFormat = time.RFC3339Nano

	logFileReqFailMsg              = "cni-log: filename is required when logging to stderr is off - will not log anything\n"
	logFileFailMsg                 = "cni-log: failed to set log file '%s'\n"
	setLevelFailMsg                = "cni-log: cannot set logging level to '%s'\n"
	symlinkEvalFailMsg             = "cni-log: unable to evaluate symbolic links on path '%v'"
	emptyStringFailMsg             = "cni-log: unable to resolve empty string"
	structuredLoggingOddArguments  = "must provide an even number of arguments for structured logging"
	structuredPrefixerOddArguments = "prefixer must return an even number of arguments for structured logging"
)

// s is the singleton holding the logger's attributes.
var s = state.Instance()

func init() {
	initLogger()
}

func initLogger() {
	s.SetLogger(&lumberjack.Logger{})

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
func SetPrefixer(p prefixer.Prefixer) {
	s.SetPrefixer(p)
}

// SetStructuredPrefixer allows overwriting the StructuredPrefixer with a custom one.
func SetStructuredPrefixer(p prefixer.StructuredPrefixer) {
	s.SetStructuredPrefixer(p)
}

// SetDefaultPrefixer sets the default Prefixer.
func SetDefaultPrefixer() {
	defaultPrefix := &prefixer.Default{
		PrefixFormat: "%s [%s] ",
		TimeFormat:   defaultTimestampFormat,
	}
	SetPrefixer(defaultPrefix)
}

// SetDefaultStructuredPrefixer sets the default StructuredPrefixer.
func SetDefaultStructuredPrefixer() {
	defaultStructuredPrefix := &prefixer.Default{
		TimeFormat: defaultTimestampFormat,
	}
	SetStructuredPrefixer(defaultStructuredPrefix)
}

// Set the logging options (LogOptions)
func SetLogOptions(o *options.Options) {
	s.SetLogOptions(o)
	// Update the logWriter if necessary.
	if s.IsFileLoggingEnabled() {
		s.SetLoggerAsLogWriter()
	}
}

// SetLogFile sets logging file.
func SetLogFile(filename string) {
	// Allow logging to stderr only. Print an error a single time when this is set to the empty string but stderr
	// logging is off.
	if filename == "" {
		if !s.GetLogToStderr() {
			fmt.Fprint(os.Stderr, logFileReqFailMsg)
		}
		s.SetLogFile("")
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

	s.SetLogFile(filename)
}

// GetLogLevel gets current logging level.
func GetLogLevel() level.Level {
	return s.GetLogLevel()
}

// SetLogLevel sets logging level.
func SetLogLevel(l level.Level) {
	if l.IsValid() {
		s.SetLogLevel(l)
	} else {
		fmt.Fprintf(os.Stderr, setLevelFailMsg, l)
	}
}

// SetLogStderr sets flag for logging stderr output.
func SetLogStderr(enable bool) {
	if !enable && !s.IsFileLoggingEnabled() {
		fmt.Fprint(os.Stderr, logFileReqFailMsg)
	}
	s.SetLogToStderr(enable)
}

// SetOutput set custom output. Warning: A subsequent call to SetLogFile or SetLogOptions invalidates this setting.
func SetOutput(out io.Writer) {
	s.SetLogWriter(out)
}

// Panicf prints logging plus stack trace. This should be used only for unrecoverable error.
func Panicf(format string, a ...interface{}) {
	printf(level.Panic, format, a...)
	printf(level.Panic, "========= Stack trace output ========")
	printf(level.Panic, "%+v", string(debug.Stack()))
	printf(level.Panic, "========= Stack trace output end ========")
}

// PanicStructured provides structured logging for log level >= panic.
func PanicStructured(msg string, args ...interface{}) {
	stackTrace := string(debug.Stack())
	args = append(args, "stacktrace", stackTrace)
	m := structuredMessage(level.Panic, msg, args...)
	printWithPrefixf(level.Panic, false, m)
}

// Errorf prints logging if logging level >= error
func Errorf(format string, a ...interface{}) error {
	printf(level.Error, format, a...)
	return fmt.Errorf(format, a...)
}

// ErrorStructured provides structured logging for log level >= error.
func ErrorStructured(msg string, args ...interface{}) error {
	m := structuredMessage(level.Error, msg, args...)
	printWithPrefixf(level.Error, false, m)
	return fmt.Errorf("%s", m)
}

// Warningf prints logging if logging level >= warning
func Warningf(format string, a ...interface{}) {
	printf(level.Warning, format, a...)
}

// WarningStructured provides structured logging for log level >= warning.
func WarningStructured(msg string, args ...interface{}) {
	m := structuredMessage(level.Warning, msg, args...)
	printWithPrefixf(level.Warning, false, m)
}

// Infof prints logging if logging level >= info
func Infof(format string, a ...interface{}) {
	printf(level.Info, format, a...)
}

// InfoStructured provides structured logging for log level >= info.
func InfoStructured(msg string, args ...interface{}) {
	m := structuredMessage(level.Info, msg, args...)
	printWithPrefixf(level.Info, false, m)
}

// Debugf prints logging if logging level >= debug
func Debugf(format string, a ...interface{}) {
	printf(level.Debug, format, a...)
}

// DebugStructured provides structured logging for log level >= debug.
func DebugStructured(msg string, args ...interface{}) {
	m := structuredMessage(level.Debug, msg, args...)
	printWithPrefixf(level.Debug, false, m)
}

// structuredMessage takes msg and an even list of args and returns a structured message.
func structuredMessage(loggingLevel level.Level, msg string, args ...interface{}) string {
	prefixArgs := s.GetStructuredPrefixer().CreateStructuredPrefix(loggingLevel, msg)
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
func printf(l level.Level, format string, a ...interface{}) {
	printWithPrefixf(l, true, format, a...)
}

// printWithPrefixf prints log messages if they match the configured log level. Messages are optionally prepended by a
// configured prefix.
func printWithPrefixf(l level.Level, printPrefix bool, format string, a ...interface{}) {
	if l > s.GetLogLevel() {
		return
	}

	if !s.IsFileLoggingEnabled() && !s.GetLogToStderr() {
		return
	}

	if printPrefix {
		format = s.GetPrefixer().CreatePrefix(l) + format
	}

	if s.GetLogToStderr() {
		doWritef(os.Stderr, format, a...)
	}

	if s.IsFileLoggingEnabled() {
		doWritef(s.GetLogWriter(), format, a...)
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
