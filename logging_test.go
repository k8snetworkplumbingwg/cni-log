package logging

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
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
	verboseMsg = "This is a VERBOSE message"
)

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CNI-LOG Test Suite")
}

var _ = Describe("CNI Logging Operations", func() {

	BeforeEach(func() {
		logLevel = defaultLogLevel
		logToStderr = false
	})

	AfterEach(func() {
		logger = &lumberjack.Logger{}

		// Setting default LogFile
		SetLogFile(defaultLogFile)
	})

	Context("Setting the log file name", func() {
		When("no logfile is provided", func() {
			It("uses the default logfile", func() {
				Expect(logger.Filename).To(Equal(defaultLogFile))
				Expect(logWriter).To(Equal(logger))
			})
		})

		When("the log file name is empty", func() {
			It("does nothing, logfile is kept as default", func() {
				expectedLoggerOutput := fmt.Sprint(emptyStringFailMsg, "")
				loggerOutput := captureStdErrStr(SetLogFile, "")

				Expect(logger.Filename).To(Equal(defaultLogFile))
				Expect(logWriter).To(Equal(logger))
				Expect(loggerOutput).To(ContainSubstring(expectedLoggerOutput))
			})
		})

		When("the log file name is valid", func() {
			filename := "/tmp/foobar.log"
			BeforeEach(func() {
				SetLogFile(filename)
			})
			AfterEach(func() {
				// Clear log file after each test run
				err := os.Remove(filename)
				Expect(err).ToNot(HaveOccurred())
			})

			It("prepares the logger's writer and creates the log file", func() {
				Expect(logWriter).To(Equal(logger))
				Expect(filename).To(BeAnExistingFile())
			})
		})

		When("the log file name is invalid", func() {
			filename := "/proc/foobar.log"
			It("ignores setting the file, logger object is default and a warning to standard output is thrown", func() {

				// Capture standard error output
				expectedLoggerOutput := fmt.Sprintf(logFileFailMsg, filename)
				loggerOutput := captureStdErrStr(SetLogFile, filename)

				Expect(logWriter).To(Equal(logger))
				Expect(filename).ToNot(BeAnExistingFile())
				Expect(logger.Filename).To(Equal(defaultLogFile))
				Expect(loggerOutput).To(Equal(expectedLoggerOutput))
			})
		})

		When("the log file is set to a symbolic link", func() {
			file := "symlink"
			symlink := "symtarget.txt"
			It("should not set logfile and keep default", func() {
				err := os.MkdirAll(file, 0755)
				if err != nil {
					Expect(err).ToNot(HaveOccurred())
				}

				err = os.Symlink(file, symlink)
				if err != nil {
					Expect(err).ToNot(HaveOccurred())
				}

				expectedLoggerOutput := fmt.Sprintf(symlinkEvalFailMsg, symlink)
				loggerOutput := captureStdErrStr(SetLogFile, symlink)

				Expect(logWriter).To(Equal(logger))
				Expect(logger.Filename).To(Equal(defaultLogFile))
				Expect(loggerOutput).To(ContainSubstring(expectedLoggerOutput))
			})
			AfterEach(func() {
				err := os.Remove(file)
				Expect(err).ToNot(HaveOccurred())
				err = os.Remove(symlink)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("Converting strings to Levels", func() {
		When("a valid string is passed", func() {
			It("returns the correct level value", func() {
				Expect(StringToLevel("warning")).To(Equal(WarningLevel))
				Expect(StringToLevel("ERROR")).To(Equal(ErrorLevel))
			})
		})

		When("an invalid string is passed", func() {
			It("returns -1", func() {
				invalidLogLevel := "invalid"
				expectedLoggerOutput := fmt.Sprintf(setLevelFailMsg, invalidLogLevel)
				loggerOutput := captureStdErrStrLev(StringToLevel, invalidLogLevel)
				Expect(loggerOutput).To(Equal(expectedLoggerOutput))
			})
		})
	})

	Context("Setting the log level", func() {
		When("a valid log level argument is passed in", func() {
			It("sets the appropriate log level", func() {
				//by string
				SetLogLevel(StringToLevel("debug"))
				Expect(logLevel).To(Equal(DebugLevel))
				SetLogLevel(StringToLevel("INFO"))
				Expect(logLevel).To(Equal(InfoLevel))
				SetLogLevel(StringToLevel("VeRbOsE"))
				Expect(logLevel).To(Equal(VerboseLevel))
				SetLogLevel(StringToLevel("warning"))
				Expect(logLevel).To(Equal(WarningLevel))
				SetLogLevel(StringToLevel("error"))
				Expect(logLevel).To(Equal(ErrorLevel))
				SetLogLevel(StringToLevel("panic"))
				Expect(logLevel).To(Equal(PanicLevel))
				//by int
				for i := 1; i <= 6; i++ {
					l := Level(i)
					SetLogLevel(l)
					Expect(logLevel).To(Equal(l))
				}
				//by level
				SetLogLevel(VerboseLevel)
				Expect(logLevel).To(Equal(VerboseLevel))
				SetLogLevel(WarningLevel)
				Expect(logLevel).To(Equal(WarningLevel))
			})
		})

		When("an invalid log level argument is passed in", func() {
			invalidLogLevel := Level(-1)
			It("maintains the current log level and logs an error", func() {
				expectedLoggerOutput := fmt.Sprintf(setLevelFailMsg, invalidLogLevel)
				loggerOutput := captureStdErrLev(SetLogLevel, invalidLogLevel)

				Expect(loggerOutput).To(Equal(expectedLoggerOutput))
				Expect(logLevel).To(Equal(defaultLogLevel))

				invalidLogLevel = Level(10)
				expectedLoggerOutput = fmt.Sprintf(setLevelFailMsg, invalidLogLevel)
				loggerOutput = captureStdErrLev(SetLogLevel, invalidLogLevel)

				Expect(loggerOutput).To(Equal(expectedLoggerOutput))
				Expect(logLevel).To(Equal(defaultLogLevel))
			})
		})
	})

	Context("Setting the log options", func() {

		logFile := "test.log"

		AfterEach(func() {
			// Clear contents of file
			data := []byte("")
			err := ioutil.WriteFile(logFile, data, 0)
			Expect(err).ToNot(HaveOccurred())
		})

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

		logFile := "test.log"

		AfterEach(func() {
			// Clear contents of file
			data := []byte("")
			err := ioutil.WriteFile(logFile, data, 0)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterAll(func() {
			// Delete test log file after all tests have run
			err := os.Remove(logFile)
			Expect(err).ToNot(HaveOccurred())
		})

		When("logfile is set to default and messages are logged", Ordered, func() {

			BeforeEach(func() {
				SetLogLevel(StringToLevel("error"))
			})

			AfterAll(func() {
				// Delete test log file after all tests have run
				err := os.Remove(defaultLogFile)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should print appropriate log messages to log file including ERROR message", func() {
				Expect(validateLogFile("error", defaultLogFile)).To(BeTrue())
			})

			It("should not log to standard output when disabled", func() {
				Expect(captureStdErrLogging(validateLogFile, "error", defaultLogFile)).To(BeEmpty())
			})

			It("should also log to standard output when enabled", func() {
				SetLogStderr(true)
				out := captureStdErrLogging(validateLogFile, "error", defaultLogFile)

				Expect(out).To(ContainSubstring(panicMsg))
				Expect(out).To(ContainSubstring(errorMsg))
				Expect(out).ToNot(ContainSubstring(infoMsg))
				Expect(out).ToNot(ContainSubstring(debugMsg))
				Expect(out).ToNot(ContainSubstring(verboseMsg))
			})
		})

		When("log level is set to ERROR and messages are logged", Ordered, func() {

			BeforeEach(func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel("error"))
			})

			It("should print appropriate log messages to log file including ERROR message", func() {
				Expect(validateLogFile("error", logFile)).To(BeTrue())
			})

			It("should not log to standard output when disabled", func() {
				Expect(captureStdErrLogging(validateLogFile, "error", logFile)).To(BeEmpty())
			})

			It("should also log to standard output when enabled", func() {
				SetLogStderr(true)
				out := captureStdErrLogging(validateLogFile, "error", logFile)

				Expect(out).To(ContainSubstring(panicMsg))
				Expect(out).To(ContainSubstring(errorMsg))
				Expect(out).ToNot(ContainSubstring(infoMsg))
				Expect(out).ToNot(ContainSubstring(debugMsg))
				Expect(out).ToNot(ContainSubstring(verboseMsg))
			})
		})

		When("log level is set to INFO and messages are logged", func() {
			BeforeEach(func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel("info"))
			})

			It("should print appropriate log messages to log file including INFO message", func() {
				Expect(validateLogFile("info", logFile)).To(BeTrue())
			})

			It("should not log to standard output when disabled", func() {
				Expect(captureStdErrLogging(validateLogFile, "info", logFile)).To(BeEmpty())
			})

			It("should also log to standard output when enabled", func() {
				SetLogStderr(true)
				out := captureStdErrLogging(validateLogFile, "info", logFile)

				Expect(out).To(ContainSubstring(panicMsg))
				Expect(out).To(ContainSubstring(errorMsg))
				Expect(out).To(ContainSubstring(infoMsg))
				Expect(out).ToNot(ContainSubstring(debugMsg))
				Expect(out).ToNot(ContainSubstring(verboseMsg))
			})
		})

		When("log level is set to VERBOSE and messages are logged", func() {
			BeforeEach(func() {
				SetLogFile(logFile)
				SetLogLevel(StringToLevel("verbose"))
			})

			It("should print appropriate log messages to log file including VERBOSE message", func() {
				Expect(validateLogFile("verbose", logFile)).To(BeTrue())
			})

			It("should not log to standard output when disabled", func() {
				Expect(captureStdErrLogging(validateLogFile, "verbose", logFile)).To(BeEmpty())
			})

			It("should also log to standard output when enabled", func() {
				SetLogStderr(true)
				out := captureStdErrLogging(validateLogFile, "verbose", logFile)

				Expect(out).To(ContainSubstring(panicMsg))
				Expect(out).To(ContainSubstring(errorMsg))
				Expect(out).To(ContainSubstring(infoMsg))
				Expect(out).To(ContainSubstring(debugMsg))
				Expect(out).To(ContainSubstring(verboseMsg))
			})
		})
	})
})

// Checks if the correct log messages are in the log file depending on the log level set
func validateLogFile(logLevel string, filename string) bool {

	logLevel = strings.ToLower(logLevel)
	logFileCorrect := true

	// Log the different log messages to file
	Panicf(panicMsg)
	_ = Errorf(errorMsg)
	Warningf(warningMsg)
	Infof(infoMsg)
	Debugf(debugMsg)
	Verbosef(verboseMsg)

	// Read in the log file
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	logContents := string(contents)

	// Validate the appropriate messages are logged in the file based on the log level set
	for levelStr, levelInt := range levelMap {
		logMessageFound := strings.Contains(logContents, levelStr)
		if !logMessageFound && levelInt <= levelMap[logLevel] {
			logFileCorrect = false
			break
		}
		if logMessageFound && levelInt > levelMap[logLevel] {
			logFileCorrect = false
			break
		}
	}

	return logFileCorrect
}

func openPipes() (*os.File, *os.File, *os.File) {
	origWriter := os.Stderr

	pipeReader, pipeWriter, err := os.Pipe() // Initialise an IO pipe
	if err != nil {
		panic(err)
	}

	os.Stderr = pipeWriter // Set stderr to point to the pipe's writer

	return pipeReader, pipeWriter, origWriter
}

func closePipes(reader *os.File, writer *os.File, orig *os.File) string {
	writer.Close()
	os.Stderr = orig // Revert stderr to what it used to be

	var buff bytes.Buffer
	_, err := io.Copy(&buff, reader) // populate a buffer with data passed in through the pipe
	if err != nil {
		panic(err) // If error is not nil then panics
	}

	return buff.String()
}

func captureStdErrLogging(f func(string, string) bool, p1 string, p2 string) string {
	pipeWriter, pipeReader, origWriter := openPipes()
	f(p1, p2)
	return closePipes(pipeWriter, pipeReader, origWriter)
}

func captureStdErrStr(f func(string), p string) string {
	pipeWriter, pipeReader, origWriter := openPipes()
	f(p)
	return closePipes(pipeWriter, pipeReader, origWriter)
}

func captureStdErrLev(f func(Level), p Level) string {
	pipeWriter, pipeReader, origWriter := openPipes()
	f(p)
	return closePipes(pipeWriter, pipeReader, origWriter)
}

func captureStdErrStrLev(f func(string) Level, p string) string {
	pipeWriter, pipeReader, origWriter := openPipes()
	f(p)
	return closePipes(pipeWriter, pipeReader, origWriter)
}

func getPrimitivePointer[P int | bool](param P) *P {
	return &param
}
