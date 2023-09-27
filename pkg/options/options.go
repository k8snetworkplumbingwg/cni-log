package options

// Options defines the configuration of the lumberjack logger.
type Options struct {
	MaxAge     *int  `json:"maxAge,omitempty"`
	MaxSize    *int  `json:"maxSize,omitempty"`
	MaxBackups *int  `json:"maxBackups,omitempty"`
	Compress   *bool `json:"compress,omitempty"`
}
