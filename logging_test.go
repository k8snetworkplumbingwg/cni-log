package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	panicMsg   = "This is a PANIC message"
	errorMsg   = "This is an ERROR message"
	warningMsg = "This is a WARNING message"
	infoMsg    = "This is an INFO message"
	debugMsg   = "This is a DEBUG message"
)

type customPrefix struct {
	prefixFormat string
	currentFile  string
}

func (cp *customPrefix) CreatePrefix(loggingLevel Level) string {
	return fmt.Sprintf(cp.prefixFormat, loggingLevel, GetLogLevel(), cp.currentFile)
}

func (cp *customPrefix) CreateStructuredPrefix(loggingLevel Level, message string) []interface{} {
	return []interface{}{
		"custom-level", loggingLevel,
		"custom-file", cp.currentFile,
		"custom-message", message,
	}
}

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CNI-LOG Test Suite")
}

var _ = Describe("CNI Logging Operations", func() {
	BeforeEach(func() {
		initLogger()
	})

	var logFile string

	BeforeEach(func() {
		logFile = path.Join(os.TempDir(), "test.log")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(logFile)).To(Succeed())
	})

	Context("Default settings", func() {
		When("the defaults are used", func() {
			It("logs to stderr", func() {
				Expect(logToStderr).To(BeTrue())
			})

			It("does not log to file", func() {
				Expect(isFileLoggingEnabled()).To(BeFalse())
			})
		})
	})

	Context("Setting error logging", func() {
		Context("File logging is disabled", func() {
			When("error logging is enabled first and file logging is disabled later", func() {
				It("does not report an error", func() {
					errStr := captureStdErr(SetLogStderr, true)
					Expect(errStr).To(BeEmpty())
					errStr = captureStdErr(SetLogFile, "")
					Expect(errStr).To(BeEmpty())
				})
			})

			When("error logging is disabled while file logging is disabled", func() {
				It("does report an error", func() {
					errStr := captureStdErr(SetLogFile, "")
					Expect(errStr).To(BeEmpty())
					errStr = captureStdErr(SetLogStderr, false)
					Expect(errStr).To(ContainSubstring(logFileReqFailMsg))
				})
			})
		})

		Context("File logging is enabled", func() {
			When("error logging is enabled first and file logging is enabled later", func() {
				It("does not report an error", func() {
					errStr := captureStdErr(SetLogStderr, true)
					Expect(errStr).To(BeEmpty())
					errStr = captureStdErr(SetLogFile, logFile)
					Expect(errStr).To(BeEmpty())
				})
			})

			When("error logging is disabled while file logging is enabled", func() {
				It("does not report an error", func() {
					errStr := captureStdErr(SetLogFile, logFile)
					Expect(errStr).To(BeEmpty())
					errStr = captureStdErr(SetLogStderr, false)
					Expect(errStr).To(BeEmpty())
				})
			})
		})
	})

	Context("Setting the log file name", func() {
		When("the log file name is empty", func() {
			It("an error to standard output is thrown when logging to stderr is off", func() {
				errStr := captureStdErr(SetLogStderr, false)
				Expect(errStr).To(ContainSubstring(logFileReqFailMsg))
				errStr = captureStdErr(SetLogFile, "")
				Expect(errStr).To(ContainSubstring(logFileReqFailMsg))
			})

			It("no error to standard output is thrown when logging to stderr is on", func() {
				errStr := captureStdErr(SetLogStderr, true)
				Expect(errStr).To(BeEmpty())
				errStr = captureStdErr(SetLogFile, "")
				Expect(errStr).To(BeEmpty())
			})
		})

		When("the log file name is valid", func() {
			It("prepares the logger's writer and creates the log file", func() {
				SetLogFile(logFile)
				Expect(logWriter).To(Equal(logger))
				Expect(logFile).To(BeAnExistingFile())
			})
		})

		When("the log file's parent directory does not exist", func() {
			var logFileDir string

			BeforeEach(func() {
				logFileDir = path.Join(os.TempDir(), "nested/nested")
				logFile = path.Join(logFileDir, "test.log")
			})

			AfterEach(func() {
				Expect(os.RemoveAll(logFileDir)).To(Succeed())
			})

			It("should be created", func() {
				SetLogFile(logFile)
				Expect(logWriter).To(Equal(logger))
				Expect(logFile).To(BeAnExistingFile())
			})
		})

		When("the log file name is invalid", func() {
			It("an error to standard output is thrown", func() {
				filename := "/proc/foobar.log"
				expectedLoggerOutput := fmt.Sprintf(logFileFailMsg, filename)
				loggerOutput := captureStdErr(SetLogFile, filename)
				Expect(loggerOutput).To(Equal(expectedLoggerOutput))
			})
		})

		When("the log file is set to a symbolic link", func() {
			var file string
			var symlink string

			BeforeEach(func() {
				tempDir := os.TempDir()
				file = path.Join(tempDir, "symlink")
				symlink = path.Join(tempDir, "symtarget.txt")

				err := os.MkdirAll(file, 0755)
				if err != nil {
					Expect(err).ToNot(HaveOccurred())
				}

				err = os.Symlink(file, symlink)
				if err != nil {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			AfterEach(func() {
				err := os.Remove(file)
				Expect(err).ToNot(HaveOccurred())
				err = os.Remove(symlink)
				Expect(err).ToNot(HaveOccurred())
			})

			It("an error to standard error is thrown", func() {
				expectedLoggerOutput := fmt.Sprintf(symlinkEvalFailMsg, symlink)
				loggerOutput := captureStdErr(SetLogFile, symlink)
				Expect(loggerOutput).To(ContainSubstring(expectedLoggerOutput))
			})
		})
	})

	Context("Setting the log options", func() {
		When("the logOption's fields are all populated", func() {
			It("logOptions should be set correctly", func() {
				expectedLogger := &lumberjack.Logger{
					Filename:   logFile,
					MaxAge:     1,
					MaxSize:    10,
					MaxBackups: 1,
					Compress:   true,
				}

				SetLogFile(logFile)
				logOpts := &LogOptions{
					MaxAge:     getPrimitivePointer(1),
					MaxSize:    getPrimitivePointer(10),
					MaxBackups: getPrimitivePointer(1),
					Compress:   getPrimitivePointer(true),
				}
				SetLogOptions(logOpts)
				Expect(logger).To(Equal(expectedLogger))
			})
		})

		When("there are some fields missing", func() {
			It("should provide default values to the missing fields", func() {
				expectedLogger := &lumberjack.Logger{
					Filename:   logFile,
					MaxAge:     5,
					MaxSize:    100,
					MaxBackups: 1,
					Compress:   true,
				}
				SetLogFile(logFile)
				logOpts := &LogOptions{
					MaxBackups: getPrimitivePointer(1),
					Compress:   getPrimitivePointer(true),
				}
				SetLogOptions(logOpts)
				Expect(logger).To(Equal(expectedLogger))
			})
		})

		When("logOptions isn't set at all", func() {
			It("should provide a default logOptions", func() {
				SetLogFile(logFile)
				expectedLogger := &lumberjack.Logger{
					Filename:   logFile,
					MaxAge:     5,
					MaxSize:    100,
					MaxBackups: 5,
					Compress:   true,
				}

				SetLogOptions(nil)
				Expect(logger).To(Equal(expectedLogger))
			})
		})
	})

	Context("Logging messages", Ordered, func() {
		When("log level is set to ERROR", Ordered, func() {
			It("should print appropriate >= error messages to log file", func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel(errorStr))
				SetLogStderr(false)

				Panicf(panicMsg)
				Expect(logFileContains(logFile, panicMsg)).To(BeTrue())
				_ = Errorf(errorMsg)
				Expect(logFileContains(logFile, errorMsg)).To(BeTrue())
				Warningf(warningMsg)
				Expect(logFileContains(logFile, warningMsg)).To(BeFalse())
				Infof(infoMsg)
				Expect(logFileContains(logFile, infoMsg)).To(BeFalse())
				Debugf(debugMsg)
				Expect(logFileContains(logFile, debugMsg)).To(BeFalse())
			})

			It("should print appropriate >= error structured messages to log file", func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel(errorStr))
				SetLogStderr(false)

				PanicStructured(panicMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, panicStr, panicMsg))).To(BeTrue())
				_ = ErrorStructured(errorMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, errorStr, errorMsg))).To(BeTrue())
				WarningStructured(warningMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, warningStr, warningMsg))).To(BeFalse())
				InfoStructured(infoMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, infoStr, infoMsg))).To(BeFalse())
				DebugStructured(debugMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, debugStr, debugMsg))).To(BeFalse())
			})
		})

		When("log level is set to INFO", func() {
			It("should print appropriate >= info messages to log file", func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel(infoStr))
				SetLogStderr(false)

				Panicf(panicMsg)
				Expect(logFileContains(logFile, panicMsg)).To(BeTrue())
				_ = Errorf(errorMsg)
				Expect(logFileContains(logFile, errorMsg)).To(BeTrue())
				Warningf(warningMsg)
				Expect(logFileContains(logFile, warningMsg)).To(BeTrue())
				Infof(infoMsg)
				Expect(logFileContains(logFile, infoMsg)).To(BeTrue())
				Debugf(debugMsg)
				Expect(logFileContains(logFile, debugMsg)).To(BeFalse())
			})

			It("should print appropriate >= info structured messages to log file", func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel(infoStr))
				SetLogStderr(false)

				PanicStructured(panicMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, panicStr, panicMsg))).To(BeTrue())
				_ = ErrorStructured(errorMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, errorStr, errorMsg))).To(BeTrue())
				WarningStructured(warningMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, warningStr, warningMsg))).To(BeTrue())
				InfoStructured(infoMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, infoStr, infoMsg))).To(BeTrue())
				DebugStructured(debugMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, debugStr, debugMsg))).To(BeFalse())
			})
		})

		When("log level is set to DEBUG and messages are logged", func() {
			It("should print appropriate >= debug messages to log file", func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel(debugStr))
				SetLogStderr(false)

				Panicf(panicMsg)
				Expect(logFileContains(logFile, panicMsg)).To(BeTrue())
				_ = Errorf(errorMsg)
				Expect(logFileContains(logFile, errorMsg)).To(BeTrue())
				Warningf(warningMsg)
				Expect(logFileContains(logFile, warningMsg)).To(BeTrue())
				Infof(infoMsg)
				Expect(logFileContains(logFile, infoMsg)).To(BeTrue())
				Debugf(debugMsg)
				Expect(logFileContains(logFile, debugMsg)).To(BeTrue())
			})

			It("should print appropriate >= debug structured messages to log file", func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel(debugStr))
				SetLogStderr(false)

				PanicStructured(panicMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, panicStr, panicMsg))).To(BeTrue())
				_ = ErrorStructured(errorMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, errorStr, errorMsg))).To(BeTrue())
				WarningStructured(warningMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, warningStr, warningMsg))).To(BeTrue())
				InfoStructured(infoMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, infoStr, infoMsg))).To(BeTrue())
				DebugStructured(debugMsg)
				Expect(logFileContainsRegex(logFile, fmt.Sprintf(`time=".*" level=%q msg=%q`, debugStr, debugMsg))).To(BeTrue())
			})
		})

		When("stucturedMessage is called with an odd number of arguments", func() {
			It("should panic", func() {
				Expect(func() { structuredMessage(InfoLevel, infoMsg, "a", "b", "c") }).Should(PanicWith(MatchRegexp( //nolint:staticcheck
					fmt.Sprintf(`^time=".*" msg=%q logging_failure=%q$`, infoMsg, structuredLoggingOddArguments))))
			})
		})

		When("custom io.Writer is set", func() {
			var out bytes.Buffer

			BeforeEach(func() {
				out = bytes.Buffer{}
				SetOutput(&out)
				SetLogStderr(false)
			})

			It("should log message to custom out", func() {
				Infof(infoMsg)
				Expect(out.String()).To(ContainSubstring(infoMsg))
			})

			It("should not log to custom out after a call to SetLogFile", func() {
				SetLogFile(logFile)
				Infof(infoMsg)
				Expect(out.String()).NotTo(ContainSubstring(infoMsg))
			})

			It("should not log to custom out after a call to SetLogOptions", func() {
				SetLogOptions(nil)
				Infof(infoMsg)
				Expect(out.String()).NotTo(ContainSubstring(infoMsg))
			})
		})

		When("error logging is on and file logging is off", func() {
			BeforeEach(func() {
				errStr := captureStdErr(SetLogStderr, true)
				Expect(errStr).To(BeEmpty())
				errStr = captureStdErr(SetLogFile, "")
				Expect(errStr).To(BeEmpty())
			})

			It("only logs to stderr", func() {
				errStr := captureStdErrEvent(Warningf, infoMsg)
				Expect(errStr).To(ContainSubstring(infoMsg))
				Expect(logFileContains(logFile, infoMsg)).To(BeFalse())
			})
		})

		When("file logging is on and error logging is off", func() {
			BeforeEach(func() {
				errStr := captureStdErr(SetLogFile, logFile)
				Expect(errStr).To(BeEmpty())
				errStr = captureStdErr(SetLogStderr, false)
				Expect(errStr).To(BeEmpty())
			})

			It("only logs to file", func() {
				errStr := captureStdErrEvent(Infof, infoMsg)
				Expect(errStr).To(BeEmpty())
				Expect(logFileContains(logFile, infoMsg)).To(BeTrue())
			})
		})

		When("file logging and error logging are turned off simultaneously", func() {
			BeforeEach(func() {
				_ = captureStdErr(SetLogFile, "")
				_ = captureStdErr(SetLogStderr, false)
			})

			It("does not log anywhere", func() {
				errStr := captureStdErrEvent(Infof, infoMsg)
				Expect(errStr).To(BeEmpty())
				Expect(logFileContains(logFile, infoMsg)).To(BeFalse())
			})
		})

		When("file logging and error logging are turned on simultaneously", func() {
			BeforeEach(func() {
				Expect(captureStdErr(SetLogFile, logFile)).To(BeEmpty())
				Expect(captureStdErr(SetLogStderr, true)).To(BeEmpty())
			})

			It("does log to both file and stderr", func() {
				errStr := captureStdErrEvent(Infof, infoMsg)
				Expect(errStr).To(ContainSubstring(infoMsg))
				Expect(logFileContains(logFile, infoMsg)).To(BeTrue())
			})
		})
	})

	Context("Updating the logging prefix", Ordered, func() {
		BeforeEach(func() {
			SetLogStderr(true)
			SetLogFile(logFile)
		})

		When("a custom prefix is not provided", func() {
			It("uses the default prefix", func() {
				expectedPrefix := fmt.Sprintf(`^.* \[%s\] `, InfoLevel)
				errStr := captureStdErrEvent(Infof, infoMsg)
				Expect(errStr).To(MatchRegexp(expectedPrefix))
				Expect(logFileContainsRegex(logFile, expectedPrefix)).To(BeTrue())
			})
		})

		When("a custom prefix is provided", func() {
			BeforeEach(func() {
				SetLogLevel(StringToLevel(debugStr))
				SetPrefixer(&customPrefix{
					prefixFormat: "[%s/%s] - %s: ",
					currentFile:  "logging_test.go",
				})
			})

			It("uses the custom prefix", func() {
				expectedPrefix := "[info/debug] - logging_test.go: "
				errStr := captureStdErrEvent(Infof, infoMsg)
				Expect(errStr).To(ContainSubstring(expectedPrefix))
				Expect(logFileContains(logFile, expectedPrefix)).To(BeTrue())
			})

			It("uses the default prefix when explicitly requesting to do so", func() {
				SetDefaultPrefixer()

				expectedPrefix := fmt.Sprintf(`^.* \[%s\] `, InfoLevel)
				errStr := captureStdErrEvent(Infof, infoMsg)
				Expect(errStr).To(MatchRegexp(expectedPrefix))
				Expect(logFileContainsRegex(logFile, expectedPrefix)).To(BeTrue())
			})
		})
	})

	Context("Updating the structured logging prefix", Ordered, func() {
		BeforeEach(func() {
			SetLogStderr(true)
			SetLogFile(logFile)
		})

		When("a custom structured prefix is not provided", func() {
			It("uses the default prefix", func() {
				expected := fmt.Sprintf(`time=".*" level=%q msg=%q`, infoStr, infoMsg)
				errStr := captureStdErrEvent(InfoStructured, infoMsg)
				Expect(errStr).To(MatchRegexp(expected))
				Expect(logFileContainsRegex(logFile, expected)).To(BeTrue())
			})
		})

		When("a custom structured prefix is provided", func() {
			BeforeEach(func() {
				SetLogLevel(StringToLevel(debugStr))
				SetStructuredPrefixer(&customPrefix{
					currentFile: "logging_test.go",
				})
			})

			It("uses the custom structured prefix", func() {
				expected := fmt.Sprintf(`custom-level=%q custom-file="logging_test.go" custom-message=%q`, infoStr, infoMsg)
				errStr := captureStdErrEvent(InfoStructured, infoMsg)
				Expect(errStr).To(MatchRegexp(expected))
				Expect(logFileContainsRegex(logFile, expected)).To(BeTrue())
			})

			It("uses the default structured prefix when explicitly requesting to do so", func() {
				SetDefaultStructuredPrefixer()

				expected := fmt.Sprintf(`time=".*" level=%q msg=%q`, infoStr, infoMsg)
				errStr := captureStdErrEvent(InfoStructured, infoMsg)
				Expect(errStr).To(MatchRegexp(expected))
				Expect(logFileContainsRegex(logFile, expected)).To(BeTrue())
			})
		})

		When("an invalid custom structured prefix is provided", func() {
			It("should panic", func() {
				var invalidPrefix StructuredPrefixerFunc = func(loggingLevel Level, message string) []interface{} {
					return []interface{}{
						"custom-level", loggingLevel,
						"custom-message", message,
						"invalid",
					}
				}
				SetStructuredPrefixer(invalidPrefix)

				Expect(func() { structuredMessage(InfoLevel, infoMsg, "a", "b", "c") }).Should(PanicWith(MatchRegexp( //nolint:staticcheck
					fmt.Sprintf(`^msg=%q logging_failure=%q$`, infoMsg, structuredPrefixerOddArguments))))
			})
		})

	})
})

var _ = Describe("CNI Log Level Operations", func() {
	BeforeEach(func() {
		initLogger()
	})

	Context("Log level", func() {
		Context("Converting strings to Levels", func() {
			When("a valid string is passed", func() {
				It("returns the correct level value", func() {
					Expect(StringToLevel(warningStr)).To(Equal(WarningLevel))
					Expect(StringToLevel("ERROR")).To(Equal(ErrorLevel))
					Expect(StringToLevel("dEbUg")).To(Equal(DebugLevel))
				})
			})

			When("an invalid string is passed", func() {
				It("returns InvalidLevel (-1)", func() {
					Expect(StringToLevel(invalidStr)).To(Equal(InvalidLevel))
				})
			})
		})

		Context("Setting the log level", func() {
			When("a valid log level argument is passed in", func() {
				It("sets the appropriate log level", func() {
					// by string
					SetLogLevel(StringToLevel(debugStr))
					Expect(logLevel).To(Equal(DebugLevel))
					SetLogLevel(StringToLevel(infoStr))
					Expect(logLevel).To(Equal(InfoLevel))
					SetLogLevel(StringToLevel(warningStr))
					Expect(logLevel).To(Equal(WarningLevel))
					SetLogLevel(StringToLevel(errorStr))
					Expect(logLevel).To(Equal(ErrorLevel))
					SetLogLevel(StringToLevel(panicStr))
					Expect(logLevel).To(Equal(PanicLevel))
					// by int
					for i := 1; i <= 5; i++ {
						l := Level(i)
						SetLogLevel(l)
						Expect(logLevel).To(Equal(l))
					}
					// by level
					SetLogLevel(DebugLevel)
					Expect(logLevel).To(Equal(DebugLevel))
					SetLogLevel(WarningLevel)
					Expect(logLevel).To(Equal(WarningLevel))
				})
			})

			When("an invalid log level argument is passed in", func() {
				invalidLogLevel := Level(-1)
				It("maintains the current log level and logs an error", func() {
					expectedLoggerOutput := fmt.Sprintf(setLevelFailMsg, invalidLogLevel)
					loggerOutput := captureStdErr(SetLogLevel, invalidLogLevel)

					Expect(loggerOutput).To(Equal(expectedLoggerOutput))
					Expect(logLevel).To(Equal(defaultLogLevel))

					invalidLogLevel = Level(10)
					expectedLoggerOutput = fmt.Sprintf(setLevelFailMsg, invalidLogLevel)
					loggerOutput = captureStdErr(SetLogLevel, invalidLogLevel)

					Expect(loggerOutput).To(Equal(expectedLoggerOutput))
					Expect(logLevel).To(Equal(defaultLogLevel))
				})
			})
		})
	})

})

// Checks if the message was logged to the log file.
func logFileContains(filename, subString string) bool {
	// Read in the log file
	contents, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	return strings.Contains(string(contents), subString)
}

// Checks if the message was logged to the log file by comparing to regular expression re.
func logFileContainsRegex(filename, re string) bool {
	// Read in the log file
	contents, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	matched, err := regexp.MatchString(re, string(contents))
	if err != nil {
		panic(err)
	}
	return matched
}

func openPipes() (*os.File, *os.File, *os.File) {
	origWriter := os.Stderr

	pipeReader, pipeWriter, err := os.Pipe() // Initialize an IO pipe
	if err != nil {
		panic(err)
	}

	os.Stderr = pipeWriter // Set stderr to point to the pipe's writer

	return pipeReader, pipeWriter, origWriter
}

func closePipes(reader, writer, orig *os.File) string {
	writer.Close()
	os.Stderr = orig // Revert stderr to what it used to be

	var buff bytes.Buffer
	_, err := io.Copy(&buff, reader) // populate a buffer with data passed in through the pipe
	if err != nil {
		panic(err) // If error is not nil then panics
	}

	return buff.String()
}

func captureStdErr[T any](f func(T), p T) string {
	pipeWriter, pipeReader, origWriter := openPipes()
	f(p)
	return closePipes(pipeWriter, pipeReader, origWriter)
}

func captureStdErrEvent(f func(string, ...interface{}), s string, a ...interface{}) string { //nolint:unparam
	pipeWriter, pipeReader, origWriter := openPipes()
	f(s, a...)
	return closePipes(pipeWriter, pipeReader, origWriter)
}

func getPrimitivePointer[P int | bool](param P) *P {
	return &param
}
