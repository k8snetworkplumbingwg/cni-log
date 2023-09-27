package prefixer

import (
	"fmt"
	"time"

	"github.com/k8snetworkplumbingwg/cni-log/pkg/level"
)

// Prefixer creator interface. Implement this interface if you wish to create a custom prefix.
type Prefixer interface {
	// Produces the prefix string. CNI-Log will call this function
	// to request for the prefix when building the logging output and will pass in the appropriate
	// log level of your log message.
	CreatePrefix(level.Level) string
}

// PrefixerFunc implements the Prefixer interface. It allows passing a function instead of a struct as the prefixer.
type PrefixerFunc func(level.Level) string

// Produces the prefix string. CNI-Log will call this function
// to request for the prefix when building the logging output and will pass in the appropriate
// log level of your log message.
func (f PrefixerFunc) CreatePrefix(logginglevel level.Level) string {
	return f(logginglevel)
}

// StructuredPrefixer creator interface. Implement this interface if you wish to to create a custom prefix for
// structured logging.
type StructuredPrefixer interface {
	// Produces the prefix string for structured logging. CNI-Log will call this function
	// to request for the prefix when building the logging output and will pass in the appropriate
	// log level of your log message.
	CreateStructuredPrefix(level.Level, string) []interface{}
}

// StructuredPrefixerFunc implements the StructuredPrefixer interface. It allows passing a function instead of a struct
// as the prefixer.
type StructuredPrefixerFunc func(level.Level, string) []interface{}

// Produces the prefix string for structured logging. CNI-Log will call this function
// to request for the prefix when building the logging output and will pass in the appropriate
// log level of your log message.
func (f StructuredPrefixerFunc) CreateStructuredPrefix(logginglevel level.Level, msg string) []interface{} {
	return f(logginglevel, msg)
}

// Default defines a default prefixer which will be used if a custom prefix is not provided. It implements both
// the Prefixer and the StructuredPrefixer interface.
type Default struct {
	PrefixFormat string
	TimeFormat   string
}

// CreatePrefix implements the Prefixer interface for the defaultPrefixer.
func (p *Default) CreatePrefix(logginglevel level.Level) string {
	return fmt.Sprintf(p.PrefixFormat, time.Now().Format(p.TimeFormat), logginglevel)
}

// CreateStructuredPrefix implements the StructuredPrefixer interface for the defaultPrefixer.
func (p *Default) CreateStructuredPrefix(logginglevel level.Level, message string) []interface{} {
	return []interface{}{
		"time", time.Now().Format(p.TimeFormat),
		"level", logginglevel,
		"msg", message,
	}
}
