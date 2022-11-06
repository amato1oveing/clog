package clog

import (
	"encoding/json"
	"fmt"
	"github.com/natefinch/lumberjack"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Format string

const (
	ConsoleFormat Format = "console"
	JsonFormat    Format = "json"
)

// Options 包含与日志相关的配置项
type Options struct {
	OutputPath  string `json:"output-path"`
	Level       Level  `json:"level"`
	Format      Format `json:"format"`
	EnableColor bool   `json:"enable-color"`
	Name        string `json:"name"`
}

// NewOptions 创建一个带有默认参数的Options对象
func NewOptions() *Options {
	return &Options{
		Level:       InfoLevel,
		Format:      ConsoleFormat,
		EnableColor: false,
		OutputPath:  ".",
		Name:        "clog",
	}
}

// Validate 验证选项字段
func (o *Options) Validate() []error {
	var errs []error

	if o.Format != ConsoleFormat && o.Format != JsonFormat {
		errs = append(errs, fmt.Errorf("not a valid log format: %q", o.Format))
	}
	if o.OutputPath == "" {
		errs = append(errs, fmt.Errorf("not a valid log outputPath: %s", o.OutputPath))
	}
	if o.Name == "" {
		errs = append(errs, fmt.Errorf("not a valid log name: %s", o.Name))
	}

	return errs
}

func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}

// Build 从配置和选项中构造一个全局的zap记录器
func (o *Options) Build() error {
	encodeLevel := zapcore.CapitalLevelEncoder
	if o.Format == ConsoleFormat && o.EnableColor {
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}
	if o.Name != "" {
		strings.Trim(o.Name, "/")
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   o.OutputPath + "/" + o.Name + "-" + time.Now().Format(TimeFormat) + ".log",
		MaxSize:    1000,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
		LocalTime:  true,
	}
	writeSyncer := zapcore.AddSync(lumberJackLogger)

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "timestamp",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: milliSecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	var encoder zapcore.Encoder
	if o.Format == ConsoleFormat {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else if o.Format == JsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	newCore := zapcore.NewCore(encoder, writeSyncer, o.Level)
	logger := zap.New(newCore, zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.PanicLevel))

	zap.RedirectStdLog(logger.Named(o.Name))
	zap.ReplaceGlobals(logger)

	return nil
}
