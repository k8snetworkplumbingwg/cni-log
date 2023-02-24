- [CNI Log](#cni-log)
- [Usage](#usage)
  - [Importing cni-log](#importing-cni-log)
  - [Customizing the logging prefix/header](#customizing-the-logging-prefixheader)
  - [Public Types \& Functions](#public-types--functions)
    - [Types](#types)
      - [Level](#level)
      - [Prefixer](#prefixer)
      - [LogOptions](#logoptions)
    - [Public setup functions](#public-setup-functions)
      - [SetLogLevel](#setloglevel)
      - [GetLogLevel](#getloglevel)
      - [StringToLevel](#stringtolevel)
      - [String](#string)
      - [SetLogStderr](#setlogstderr)
      - [SetLogOptions](#setlogoptions)
      - [SetLogFile](#setlogfile)
      - [SetOutput](#setoutput)
      - [SetPrefixer](#setprefixer)
      - [SetDefaultPrefixer](#setdefaultprefixer)
    - [Logging functions](#logging-functions)
  - [Default values](#default-values)

## CNI Log

The purpose of this package is to perform logging for CNI projects in NPWG. Cni-log provides general logging functionality for Container Network Interfaces (CNI). Messages can be logged to a log file and/or to standard error.  

## Usage

The package can function out of the box as most of its configurations have [default values](#default-values). Just call any of the [logging functions](#logging-functions) to start logging. To further define log settings such as the log file path, the log level, as well as the lumberjack logger object, continue on to the [public functions below](#public-types--functions).

### Importing cni-log

Import cni-log in your go file:

```go
import (
    ...
    "github.com/k8snetworkplumbingwg/cni-log"
    ...
)
```

Then perform a `go mod tidy` to download the package.

### Customizing the logging prefix/header

CNI-log allows users to modify the logging prefix/header. The default prefix is in the following format:

```
yyyy-mm-ddTHH:MM:SSZ [<log level>] ...
```

E.g.

```
2022-10-11T13:09:57Z [info] This is a log message with INFO log level
```

To change the prefix used by cni-log, you will need to provide the implementation of how the prefix string would be built. To do so you will need to create an object of type [``Prefixer``](#prefixer). ``Prefixer`` is an interface with one function:

```go
CreatePrefix(Level) string
```

Implement the above function with the code that would build the prefix string. In order for CNI-Log to use your custom prefix you will then need to pass in your custom prefix object using the [``SetPrefixer``](#setprefixer) function.

Below is sample code on how to build a custom prefix:

```go
package main
import (
  ...
  logging "github.com/k8snetworkplumbingwg/cni-log"
  ...
)

// custom prefix type
type customPrefix struct {
  prefixFormat string
  timeFormat   string
  currentFile  string
}

func main() {
  // cni-log configuration
  logging.SetLogFile("samplelog.log")
  logging.SetLogLevel(logging.VerboseLevel)
  logging.SetLogStderr(true)

  // Creating the custom prefix object
  prefix := &customPrefix{
    prefixFormat: "%s | %s | %s | ",
    timeFormat:   time.RFC850,
    currentFile:  "main.go",
  }
  logging.SetPrefixer(prefix) // Tell cni-log to use your custom prefix object

  // Log messages
  logging.Infof("Info log message")
  logging.Warningf("Warning log message")
}

// Implement the CreatePrefix function using your custom prefix object. This function will be called by CNI-Log
// to build the prefix string. 
func (p *customPrefix) CreatePrefix(loggingLevel logging.Level) string {
  currentTime := time.Now()
  return fmt.Sprintf(p.prefixFormat, currentTime.Format(p.timeFormat), p.currentFile, loggingLevel)
}
```

### Public Types & Functions

#### Types

##### Level

```go
// Level type
type Level int
```

Defines the type that will represent the different log levels

##### Prefixer

```go
type Prefixer interface {
  CreatePrefix(Level) string
}
```

Defines an interface that contains one function: ``CreatePrefix(Level) string``. Implementing this function allows you to build your own custom prefix.

##### LogOptions

```go
// LogOptions defines the configuration of the lumberjack logger
type LogOptions struct {
  MaxAge     *int  `json:"maxAge,omitempty"`
  MaxSize    *int  `json:"maxSize,omitempty"`
  MaxBackups *int  `json:"maxBackups,omitempty"`
  Compress   *bool `json:"compress,omitempty"`
}
```

For further details of each field, see the [lumberjack documentation](https://github.com/natefinch/lumberjack).

To view the default values of each field, go to the "[Default values](#default-values)" section

#### Public setup functions

##### SetLogLevel

```go
func SetLogLevel(level Level)
```

Sets the log level. The valid log levels are:
| int | string | Level |
| --- | --- | --- |
| 1 | panic | PanicLevel |
| 2 | error | ErrorLevel |
| 3 | warning | WarningLevel |
| 4 | info | InfoLevel |
| 5 | debug | DebugLevel |
| 6 | verbose | VerboseLevel |

The log levels above are in ascending order of verbosity. For example, setting the log level to InfoLevel would mean "panic", "error", warning", and "info" messages will get logged while "debug", and "verbose" will not.

##### GetLogLevel

```go
func GetLogLevel() Level
```

Returns the current log level

##### StringToLevel

```go
func StringToLevel(level string) Level
```

Returns the Level equivalent of a string. See SetLogLevel for valid levels.

##### String

```go
func (l Level) String() string
```

Returns the string representation of a log level

##### SetLogStderr

```go
func SetLogStderr(enable bool)
```

This function allows you to enable/disable logging to standard error.

##### SetLogOptions

```go
func SetLogOptions(options *LogOptions)
```

Configures the lumberjack object based on the lumberjack configuration data set in the ``logOptions`` object (see ``logOptions`` struct above).

##### SetLogFile

```go
func SetLogFile(filename string)
```

Configures where logs will be written to. If an empty/invalid filepath (e.g. insufficient permissions), or a symbolic link is passed into the function, the default log filepath is used.

##### SetOutput

```go
func SetOutput(out io.Writer)
```

Set custom output. Calling this function will discard any previously set LogOptions.

##### SetPrefixer

```go
func SetPrefixer(p Prefixer)
```

This function allows you to override the default logging prefix with a custom prefix.

##### SetDefaultPrefixer

```go
func SetDefaultPrefixer()
```

This function allows you to return to the default logging prefix.

#### Logging functions

```go
// Errorf prints logging if logging level >= error
func Errorf(format string, a ...interface{}) error 

// Warningf prints logging if logging level >= warning
func Warningf(format string, a ...interface{})

// Infof prints logging if logging level >= info
func Infof(format string, a ...interface{})

// Debugf prints logging if logging level >= debug
func Debugf(format string, a ...interface{})

// Verbosef prints logging if logging level >= verbose
func Verbosef(format string, a ...interface{})
```

### Default values

| Variable | Default Value |
| ---     | ---           |
| logLevel | info |
| Logger.Filename | ``/var/log/cni-log.log`` |
| LogOptions.MaxSize | 100 |
| LogOptions.MaxAge | 5 |
| LogOptions.MaxBackups | 5 |
| LogOptions.Compress | true |
