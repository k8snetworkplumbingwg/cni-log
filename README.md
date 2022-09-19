- [CNI Log](#cni-log)
- [Usage](#usage)
  - [Importing cni-log](#importing-cni-log)
  - [Public Types & Functions](#public-types--functions)
    - [Types](#types)
      - [Level](#level)
      - [LogOptions](#logoptions)
    - [Public setup functions](#public-setup-functions)
      - [SetLogLevel](#setloglevel)
      - [GetLogLevel](#getloglevel)
      - [String](#string)
      - [SetLogStderr](#setlogstderr)
      - [SetLogOptions](#setlogoptions)
      - [SetLogFile](#setlogfile)
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

### Public Types & Functions
#### Types
##### Level
```go
// Level type
type Level uint8
```
Defines the type that will represent the different log levels

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
func SetLogLevel(level string)
```
Sets the log level. The valid log levels are:
- panic 
- error 
- warning 
- info 
- debug
- verbose

The log levels above are in ascending order of verbosity. For example setting the log level to "info" would mean "panic", "error", warning", and "info" messages will get logged while "debug", and "verbose" will not. 

##### GetLogLevel
```go
func GetLogLevel() Level
```
Returns the current log level

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