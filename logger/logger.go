package logger

import (
	"os"

	"github.com/Qsnh/backup-pro/config"
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// InitLogger 初始化日志系统
func InitLogger(cfg config.LoggingConfig) error {
	Log = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)

	// 设置日志格式
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 设置输出
	if cfg.Output == "file" && cfg.File != "" {
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		Log.SetOutput(file)
	} else {
		Log.SetOutput(os.Stdout)
	}

	return nil
}

// GetLogger 获取日志实例
func GetLogger() *logrus.Logger {
	if Log == nil {
		Log = logrus.New()
		Log.SetLevel(logrus.InfoLevel)
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}
	return Log
}
