package server

import (
	"time"
)

// Config holds info required to configure a Server.Server.
type Config struct {
	// MaxHeaderBytes can be used to override the default of 1<<20.
	MaxHeaderBytes int `json:"max_header_bytes"`

	// ReadTimeout can be used to override the default http Server timeout of 20s.
	// The string should be formatted like a time.Duration string.
	ReadTimeout time.Duration `json:"read_timeout"`

	// WriteTimeout can be used to override the default http Server timeout of 20s.
	// The string should be formatted like a time.Duration string.
	WriteTimeout time.Duration `json:"write_timeout"`

	// IdleTimeout can be used to override the default http Server timeout of 120s.
	// The string should be formatted like a time.Duration string.
	IdleTimeout time.Duration `json:"idle_timeout"`

	// ShutdownTimeout can be used to override the default http Server Shutdown timeout
	// of 5m.
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// HTTPPort is the port the Server implementation will serve HTTP over.
	// The default is 8080
	HTTPPort int `json:"http_port"`

	// RPCPort is the port the Server implementation will serve RPC over.
	// The default is 8081.
	RPCPort int `json:"rpc_port"`

	// Enable pprof Profiling. Off by default.
	EnablePProf bool `json:"enable_pprof"`

	// LoggerHandler level (eg.: panic, fatal, error, warn, info, debug)
	LoggerLevel string `json:"logger_level"`

	// LoggerHandler format (ex.: text, json)
	LoggerFormat string `json:"logger_format"`
}

// DefaultConfig returns a generic Server configuration.
func DefaultConfig() Config {
	return Config{
		MaxHeaderBytes:  1 << 20,
		ReadTimeout:     20 * time.Second,
		WriteTimeout:    20 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 5 * time.Minute,
		HTTPPort:        8080,
		RPCPort:         8081,
		EnablePProf:     false,
		LoggerLevel:     "info",
		LoggerFormat:    "json",
	}
}
