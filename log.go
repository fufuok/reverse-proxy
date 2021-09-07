package rproxy

import (
	"github.com/phuslu/log"
)

func initLogger() {
	log.DefaultLogger = log.Logger{
		Level:      log.ParseLevel(conf.LogLevel),
		TimeFormat: "0102 15:04:05",
		Writer: &log.MultiWriter{
			InfoWriter: &log.FileWriter{
				Filename:     conf.LogFile,
				FileMode:     0600,
				MaxSize:      100 << 20,
				MaxBackups:   7,
				EnsureFolder: true,
				LocalTime:    true,
			},
			ErrorWriter: &log.FileWriter{
				Filename:     conf.ErrorLogFile,
				FileMode:     0600,
				MaxSize:      100 << 20,
				MaxBackups:   30,
				EnsureFolder: true,
				LocalTime:    true,
			},
		},
	}
	if conf.Debug {
		log.DefaultLogger.Writer = &log.ConsoleWriter{
			ColorOutput:    true,
			QuoteString:    true,
			EndWithMessage: true,
		}
	}
}
