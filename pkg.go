// Package neith brings enhanced functionality to the Component interface.
package neith

import (
	"math"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var config *Config

var logOpts = log.Options{
	ReportCaller:    true,
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "package neith:",
}

type LogLevel log.Level

const (
	Debug LogLevel = -4
	Info  LogLevel = 0
	Warn  LogLevel = 4
	Error LogLevel = 8
	Fatal LogLevel = 12
	None  LogLevel = math.MaxInt32
)

func init() {
	config = defaultConfig()
	defaultRuntime = newRuntime(config)
}

func defaultConfig() *Config {
	return &Config{
		CacheTimeOut:    time.Minute * 30,
		LogLevel:        Error,
		Logger:          log.NewWithOptions(os.Stderr, logOpts),
		UploadDir:       "",
		UploadMaxBytes:  64 << 20,
		UploadMaxMemory: 32 << 20,
	}
}

type Config struct {
	Silent          bool          // If true, no logs will be printed
	CacheTimeOut    time.Duration // Default cache timeout
	LogLevel        LogLevel
	Logger          *log.Logger
	UploadDir       string // Directory for multipart event uploads; defaults to os.TempDir()/neith-uploads
	UploadMaxBytes  int64  // Maximum request size for one upload request
	UploadMaxMemory int64  // Maximum multipart memory before files spill to disk
}

func SetConfig(c *Config) {
	config = c
	config.Set()
	if defaultRuntime != nil {
		defaultRuntime.config = config
	}
}

func (c *Config) Set() {
	if c.Logger == nil {
		c.Logger = log.NewWithOptions(os.Stderr, logOpts)
	}

	config = c
	if c.Silent || c.LogLevel == None {
		c.Logger.SetLevel(log.Level(None))
		return
	}
	c.Logger.Info(
		"neith config set",
		"cache_timeout", c.CacheTimeOut,
		"log_level", c.LogLevel,
	)

	config.Logger.SetLevel(log.Level(c.LogLevel))
}
