package logger

import (
	"context"
	"fmt"
	"os"

	kratoslog "github.com/go-kratos/kratos/v2/log"
)

// KratosLogger Kratos日志适配器
type KratosLogger struct {
	logger Logger
}

// NewKratosLogger 创建Kratos日志适配器
func NewKratosLogger(logger Logger) kratoslog.Logger {
	return &KratosLogger{logger: logger}
}

// Log 实现Kratos Logger接口
func (kl *KratosLogger) Log(level kratoslog.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}

	// 解析键值对
	fields := make(map[string]interface{})
	var msg string

	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key := fmt.Sprintf("%v", keyvals[i])
			value := keyvals[i+1]

			if key == "msg" {
				msg = fmt.Sprintf("%v", value)
			} else {
				fields[key] = value
			}
		}
	}

	// 转换日志级别并记录
	ctx := context.TODO()
	switch level {
	case kratoslog.LevelDebug:
		kl.logger.Debug(ctx, msg, convertFields(fields)...)
	case kratoslog.LevelInfo:
		kl.logger.Info(ctx, msg, convertFields(fields)...)
	case kratoslog.LevelWarn:
		kl.logger.Warn(ctx, msg, convertFields(fields)...)
	case kratoslog.LevelError:
		kl.logger.Error(ctx, msg, convertFields(fields)...)
	case kratoslog.LevelFatal:
		kl.logger.Fatal(ctx, msg, convertFields(fields)...)
	default:
		kl.logger.Info(ctx, msg, convertFields(fields)...)
	}

	return nil
}

// convertFields 转换字段格式
func convertFields(fields map[string]interface{}) []Field {
	result := make([]Field, 0, len(fields))
	for key, value := range fields {
		result = append(result, F(key, value))
	}
	return result
}

// NewKratosStdLogger 创建标准的Kratos日志器
func NewKratosStdLogger(serviceName, version string) kratoslog.Logger {
	return kratoslog.With(
		kratoslog.NewStdLogger(os.Stdout),
		"service.name", serviceName,
		"service.version", version,
		"ts", kratoslog.DefaultTimestamp,
		"caller", kratoslog.DefaultCaller,
	)
}

// NewKratosFileLogger 创建文件输出的Kratos日志器
func NewKratosFileLogger(serviceName, version, filename string) (kratoslog.Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return kratoslog.With(
		kratoslog.NewStdLogger(file),
		"service.name", serviceName,
		"service.version", version,
		"ts", kratoslog.DefaultTimestamp,
		"caller", kratoslog.DefaultCaller,
	), nil
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	ServiceName string
	Version     string
	Level       string
	Output      string // "stdout", "file", "both"
	Filename    string
}

// NewKratosLoggerWithConfig 根据配置创建Kratos日志器
func NewKratosLoggerWithConfig(config LoggerConfig) (kratoslog.Logger, error) {
	var loggers []kratoslog.Logger

	switch config.Output {
	case "stdout":
		loggers = append(loggers, NewKratosStdLogger(config.ServiceName, config.Version))
	case "file":
		if config.Filename == "" {
			return nil, fmt.Errorf("filename is required for file output")
		}
		fileLogger, err := NewKratosFileLogger(config.ServiceName, config.Version, config.Filename)
		if err != nil {
			return nil, err
		}
		loggers = append(loggers, fileLogger)
	case "both":
		loggers = append(loggers, NewKratosStdLogger(config.ServiceName, config.Version))
		if config.Filename != "" {
			fileLogger, err := NewKratosFileLogger(config.ServiceName, config.Version, config.Filename)
			if err != nil {
				return nil, err
			}
			loggers = append(loggers, fileLogger)
		}
	default:
		loggers = append(loggers, NewKratosStdLogger(config.ServiceName, config.Version))
	}

	if len(loggers) == 1 {
		return loggers[0], nil
	}

	// 如果有多个日志器，创建组合日志器
	return &multiLogger{loggers: loggers}, nil
}

// multiLogger 多路日志器
type multiLogger struct {
	loggers []kratoslog.Logger
}

// Log 实现Kratos Logger接口，向所有日志器输出
func (ml *multiLogger) Log(level kratoslog.Level, keyvals ...interface{}) error {
	for _, logger := range ml.loggers {
		if err := logger.Log(level, keyvals...); err != nil {
			// 继续尝试其他日志器，不因为一个失败而停止
			continue
		}
	}
	return nil
}
