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
	"time"
)

// Prefixer creator interface. Implement this interface if you wish to create a custom prefix.
type Prefixer interface {
	// Produces the prefix string. CNI-Log will call this function
	// to request for the prefix when building the logging output and will pass in the appropriate
	// log level of your log message.
	CreatePrefix(Level) string
}

// PrefixerFunc implements the Prefixer interface. It allows passing a function instead of a struct as the prefixer.
type PrefixerFunc func(Level) string

// Produces the prefix string. CNI-Log will call this function
// to request for the prefix when building the logging output and will pass in the appropriate
// log level of your log message.
func (f PrefixerFunc) CreatePrefix(loggingLevel Level) string {
	return f(loggingLevel)
}

// StructuredPrefixer creator interface. Implement this interface if you wish to to create a custom prefix for
// structured logging.
type StructuredPrefixer interface {
	// Produces the prefix string for structured logging. CNI-Log will call this function
	// to request for the prefix when building the logging output and will pass in the appropriate
	// log level of your log message.
	CreateStructuredPrefix(Level, string) []interface{}
}

// StructuredPrefixerFunc implements the StructuredPrefixer interface. It allows passing a function instead of a struct
// as the prefixer.
type StructuredPrefixerFunc func(Level, string) []interface{}

// Produces the prefix string for structured logging. CNI-Log will call this function
// to request for the prefix when building the logging output and will pass in the appropriate
// log level of your log message.
func (f StructuredPrefixerFunc) CreateStructuredPrefix(loggingLevel Level, msg string) []interface{} {
	return f(loggingLevel, msg)
}

// Defines a default prefixer which will be used if a custom prefix is not provided. It implements both the Prefixer
// and the StructuredPrefixer interface.
type defaultPrefixer struct {
	prefixFormat string
	timeFormat   string
}

// CreatePrefix implements the Prefixer interface for the defaultPrefixer.
func (p *defaultPrefixer) CreatePrefix(loggingLevel Level) string {
	return fmt.Sprintf(p.prefixFormat, time.Now().Format(p.timeFormat), loggingLevel)
}

// CreateStructuredPrefix implements the StructuredPrefixer interface for the defaultPrefixer.
func (p *defaultPrefixer) CreateStructuredPrefix(loggingLevel Level, message string) []interface{} {
	return []interface{}{
		"time", time.Now().Format(p.timeFormat),
		"level", loggingLevel,
		"msg", message,
	}
}
