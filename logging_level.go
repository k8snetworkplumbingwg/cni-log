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
	InvalidLevel Level = -1
	PanicLevel   Level = 1
	ErrorLevel   Level = 2
	WarningLevel Level = 3
	InfoLevel    Level = 4
	DebugLevel   Level = 5
	maximumLevel Level = DebugLevel

	panicStr   = "panic"
	errorStr   = "error"
	warningStr = "warning"
	infoStr    = "info"
	debugStr   = "debug"
	invalidStr = "invalid"
)

var levelMap = map[string]Level{
	panicStr:   PanicLevel,
	errorStr:   ErrorLevel,
	warningStr: WarningLevel,
	infoStr:    InfoLevel,
	debugStr:   DebugLevel,
}

// Level type
type Level int

// String converts a Level into its string representation.
func (l Level) String() string {
	switch l {
	case PanicLevel:
		return panicStr
	case WarningLevel:
		return warningStr
	case InfoLevel:
		return infoStr
	case ErrorLevel:
		return errorStr
	case DebugLevel:
		return debugStr
	case InvalidLevel:
		return invalidStr
	default:
		return invalidStr
	}
}

func StringToLevel(level string) Level {
	if l, found := levelMap[strings.ToLower(level)]; found {
		return l
	}
	return InvalidLevel
}

func validateLogLevel(level Level) bool {
	return level > 0 && level <= maximumLevel
}
