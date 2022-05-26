package logger

import (
	"data-sync-agent/helper"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Log is intended as global logger instance pre-initialized by the
// framework
func Log() *zap.Logger {
	if log == nil {
		tmpLog, _ := zap.NewDevelopment()
		return tmpLog
	}
	return log
}

var atom = zap.NewAtomicLevel()

var state = zap.String("state", "bootstrapping")

func setLog(level zapcore.Level, encoding string) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.StacktraceKey = "stack"
	cfg.EncoderConfig.CallerKey = "line"
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	cfg.Encoding = encoding
	cfg.Level = atom
	if encoding == "console" {
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := cfg.Build(zap.AddCaller())

	if err != nil {
		// fmt.Println(`{ "level": "error", "msg":  }`)
		if log != nil {
			log.With(zap.Error(err)).Warn("New settings not applied.")
		}
		return
	}

	// By Default log everything.
	atom.SetLevel(level)

	if log != nil {
		log.Sync()
	}

	log = logger // .Named(theService.Name) // .With(state)

	// if theService != nil {
	// 	log = log.Named(theService.Name)
	// }
}

// func init() {
// 	// setLog(zap.DebugLevel, "json")
// }

// Struct to hold log settings
type loggingSetting struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

func stringToLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.DebugLevel
	}

	return zap.DebugLevel
}

// Singleton setting to hold global log related settings
var logSetting = loggingSetting{"debug", "text"}

// This is kind of a mapping function which maps which returns
// Ref to settings
func loggingSettingFactory(setting string) interface{} {
	if setting == "/level" {
		// Ref to level, the value will be automatically copied
		return &logSetting.Level
	} else if setting == "/format" {
		// Ref to format, the value will be automatically copied
		return &logSetting.Format
	}
	return nil
}

// This function has to be called when one wants to apply log settings
func applySetting(s *loggingSetting) {

	format := s.Format
	if format == "text" {
		format = "console"
	} else if format != "json" {
		if log != nil {
			log.Warn("Invalid log format provider, log settings not applied")
		}
		return
	}

	setLog(stringToLogLevel(s.Level), format)
}

// LogInitializer helps modules initialize its logger
// type LogInitializer func(log *zap.Logger)

// DevSettings represents developer settings like debug mode
// local discovery etc.
type DevSettings struct {
	Debug     bool
	Discovery map[string]string
}

func init() {

	//initializing logger with format and log-level
	logLevel := helper.GetEnv(helper.LoggerLogLevel)
	logFormat := helper.GetEnv(helper.LoggerLogFormat)
	setLog(stringToLogLevel(logLevel), logFormat)
}
