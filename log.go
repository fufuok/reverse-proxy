package rproxy

import (
	"time"

	"github.com/phuslu/log"
)

func initLogger() {
	if conf.Debug {
		log.DefaultLogger = log.Logger{
			Level:      log.ParseLevel(conf.LogLevel),
			TimeFormat: "0102 15:04:05",
			Writer: &log.ConsoleWriter{
				ColorOutput:    true,
				QuoteString:    true,
				EndWithMessage: true,
			},
		}
		return
	}

	log.DefaultLogger = log.Logger{
		Level:      log.ParseLevel(conf.LogLevel),
		TimeFormat: time.RFC3339,
		Writer: &log.MultiWriter{
			InfoWriter: &log.FileWriter{
				Filename:     conf.LogFile,
				FileMode:     0600,
				MaxSize:      100 << 20,
				MaxBackups:   7,
				EnsureFolder: true,
				LocalTime:    true,
				ProcessID:    true,
			},
			ErrorWriter: &log.FileWriter{
				Filename:     conf.ErrorLogFile,
				FileMode:     0600,
				MaxSize:      100 << 20,
				MaxBackups:   30,
				EnsureFolder: true,
				LocalTime:    true,
				ProcessID:    true,
			},
		},
	}
}
