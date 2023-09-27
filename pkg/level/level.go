package level

import "strings"

/*
Common use of different level:

"panic":   Code crash
"error":   Unusual event occurred (invalid input or system issue), so exiting code prematurely
"warning": Unusual event occurred (invalid input or system issue), but continuing
"info":    Basic information, indication of major code paths
"debug":   Additional information, indication of minor code branches
*/

const (
	Invalid Level = -1
	Panic   Level = 1
	Error   Level = 2
	Warning Level = 3
	Info    Level = 4
	Debug   Level = 5
	maximum Level = Debug

	panicStr   = "panic"
	errorStr   = "error"
	warningStr = "warning"
	infoStr    = "info"
	debugStr   = "debug"
	invalidStr = "invalid"
)

var levelMap = map[string]Level{
	panicStr:   Panic,
	errorStr:   Error,
	warningStr: Warning,
	infoStr:    Info,
	debugStr:   Debug,
}

// Parse converts the provided string to a log Level. If the provided level is invalid, it returns InvalidLevel.
func Parse(level string) Level {
	if l, found := levelMap[strings.ToLower(level)]; found {
		return l
	}
	return Invalid
}

// Level type
type Level int

// String converts a Level into its string representation.
func (l Level) String() string {
	switch l {
	case Panic:
		return panicStr
	case Warning:
		return warningStr
	case Info:
		return infoStr
	case Error:
		return errorStr
	case Debug:
		return debugStr
	case Invalid:
		return invalidStr
	default:
		return invalidStr
	}
}

// IsValid checks if the provided log level is correct.
func (l Level) IsValid() bool {
	return l > 0 && l <= maximum
}
