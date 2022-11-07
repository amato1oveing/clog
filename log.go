package clog

import (
	"context"
	"fmt"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"strings"
	"sync"
	"time"
)

// InfoLogger 是一个只能打印Info信息的Logger接口
type InfoLogger interface {
	Info(msg string, fields ...Field)
	Infof(format string, v ...interface{})
	Infow(msg string, keysAndValues ...interface{})

	// Enabled 测试InfoLogger是否已启用
	Enabled() bool
}

// noopInfoLogger 是一个InfoLogger实现，但是什么也不做
type noopInfoLogger struct{}

func (l *noopInfoLogger) Enabled() bool                    { return false }
func (l *noopInfoLogger) Info(_ string, _ ...Field)        {}
func (l *noopInfoLogger) Infof(_ string, _ ...interface{}) {}
func (l *noopInfoLogger) Infow(_ string, _ ...interface{}) {}

// Logger 一个可以打印各种类型信息的接口
type Logger interface {
	InfoLogger
	Debug(msg string, fields ...Field)
	Debugf(format string, v ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Warn(msg string, fields ...Field)
	Warnf(format string, v ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Error(msg string, fields ...Field)
	Errorf(format string, v ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Panic(msg string, fields ...Field)
	Panicf(format string, v ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Fatal(msg string, fields ...Field)
	Fatalf(format string, v ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	// WithValues 添加一些上下文键值对
	WithValues(keysAndValues ...interface{}) Logger

	// WithName 为日志记录器的名称添加一个新元素
	// 连续的WithName调用继续追加在后缀
	// 名称段只包含字母、数字和连字符
	WithName(name string) Logger

	// Flush 调用底层核心的Sync方法，刷新所有缓冲
	// 应用程序在退出前应注意调用Sync
	Flush()
}

var _ Logger = &zapLogger{}

// infoLogger是一个InfoLogger实现，由Zap实现
type infoLogger struct {
	level zapcore.Level
	log   *zap.Logger
}

func (l *infoLogger) Enabled() bool { return true }

func (l *infoLogger) Info(msg string, fields ...Field) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(fields...)
	}
}

func (l *infoLogger) Infof(format string, args ...interface{}) {
	if checkedEntry := l.log.Check(l.level, fmt.Sprintf(format, args...)); checkedEntry != nil {
		checkedEntry.Write()
	}
}

func (l *infoLogger) Infow(msg string, keysAndValues ...interface{}) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(handleFields(l.log, keysAndValues)...)
	}
}

// zapLogger是一个Logger实现
type zapLogger struct {
	zapLogger *zap.Logger
	infoLogger
	opts             *Options
	lumberJackLogger *lumberjack.Logger
	contextKeys      []string
	day              int
}

//handleFields 将一些key/value形式的错误信息转化为zap中的Field格式
func handleFields(l *zap.Logger, args []interface{}, additional ...zap.Field) []zap.Field {
	// a slightly modified version of zap.SugaredLogger.sweetenFields
	if len(args) == 0 {
		// fast-return if we have no suggared fields.
		return additional
	}

	// unlike Zap, we can be pretty sure users aren't passing structured
	// fields (since logr has no concept of that), so guess that we need a
	// little less space.
	fields := make([]zap.Field, 0, len(args)/2+len(additional))
	for i := 0; i < len(args); {
		// check just in case for strongly-typed Zap fields, which is illegal (since
		// it breaks implementation agnosticism), so we can give a better error message.
		if _, ok := args[i].(zap.Field); ok {
			l.DPanic("strongly-typed Zap Field passed to logr", zap.Any("zap field", args[i]))

			break
		}

		// make sure this isn't a mismatched key
		if i == len(args)-1 {
			l.DPanic("odd number of arguments passed as key-value pairs for logging", zap.Any("ignored key", args[i]))

			break
		}

		// process a key-value pair,
		// ensuring that the key is a string
		key, val := args[i], args[i+1]
		keyStr, isString := key.(string)
		if !isString {
			// if the key isn't a string, DPanic and stop logging
			l.DPanic(
				"non-string key argument passed to logging, ignoring all later arguments",
				zap.Any("invalid key", key),
			)

			break
		}

		fields = append(fields, zap.Any(keyStr, val))
		i += 2
	}

	return append(fields, additional...)
}

var (
	std = New(NewOptions())
	mu  sync.Mutex
)

// Init 使用Options初始化std
func Init(opts *Options) {
	mu.Lock()
	defer mu.Unlock()
	std = New(opts)
}

// New 通过Options创建Logger
func New(opts *Options) *zapLogger {
	if opts == nil {
		opts = NewOptions()
	}

	encodeLevel := zapcore.CapitalLevelEncoder
	if opts.Format == ConsoleFormat && opts.EnableColor {
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}
	if opts.Name != "" {
		strings.Trim(opts.Name, "/")
	}

	lumberJackLogger := opts.GetLumberJackLogger()
	writeSyncer := zapcore.AddSync(lumberJackLogger)

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "timestamp",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		FunctionKey:    zapcore.OmitKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	var encoder zapcore.Encoder
	if opts.Format == ConsoleFormat {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else if opts.Format == JsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	newCore := zapcore.NewCore(encoder, writeSyncer, opts.Level)
	l := zap.New(newCore, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.PanicLevel))

	logger := &zapLogger{
		zapLogger: l.Named(opts.Name),
		infoLogger: infoLogger{
			log:   l,
			level: zap.InfoLevel,
		},
		opts:             opts,
		lumberJackLogger: lumberJackLogger,
		contextKeys:      make([]string, 0),
		day:              time.Now().Day(),
	}
	zap.RedirectStdLog(l)
	zap.ReplaceGlobals(l)

	return logger
}

// ZapLogger 返回全局的ZapLogger
func ZapLogger() *zap.Logger { return std.zapLogger }

//ZapLogger 返回ZapLogger
func (l *zapLogger) ZapLogger() *zap.Logger { return l.zapLogger }

// LumberJackLogger 返回全局的lumberJackLogger
func LumberJackLogger() *lumberjack.Logger { return std.lumberJackLogger }

// LumberJackLogger 返回lumberJackLogger
func (l *zapLogger) LumberJackLogger() *lumberjack.Logger { return l.lumberJackLogger }

// ContextKeys 返回全局的contextKeys
func ContextKeys() []string { return std.contextKeys }

// ContextKeys 返回contextKeys
func (l *zapLogger) ContextKeys() []string { return l.contextKeys }

// SugaredLogger 返回全局的SugaredLogger
func SugaredLogger() *zap.SugaredLogger { return std.zapLogger.Sugar() }

// SugaredLogger 返回SugaredLogger
func (l *zapLogger) SugaredLogger() *zap.SugaredLogger { return l.zapLogger.Sugar() }

// StdErrLogger 返回标准库Logger，错误级别为error
func StdErrLogger() *log.Logger {
	if std == nil {
		return nil
	}
	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.ErrorLevel); err == nil {
		return l
	}

	return nil
}

// StdInfoLogger 返回标准库Logger，错误级别为info
func StdInfoLogger() *log.Logger {
	if std == nil {
		return nil
	}
	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.InfoLevel); err == nil {
		return l
	}

	return nil
}

// WithValues 创建一个子Logger，并将字段添加到其中
func WithValues(keysAndValues ...interface{}) Logger { return std.WithValues(keysAndValues...) }

//WithValues 创建一个子Logger，并将字段添加到其中
func (l *zapLogger) WithValues(keysAndValues ...interface{}) Logger {
	newLogger := l.zapLogger.With(handleFields(l.zapLogger, keysAndValues)...)

	return NewLogger(newLogger, l.lumberJackLogger, l.opts)
}

// WithName 为日志记录器的名称添加一个新元素
// 连续的WithName调用继续追加在后缀
// 名称段只包含字母、数字和连字符
func WithName(s string) Logger { return std.WithName(s) }

// WithName 为日志记录器的名称添加一个新元素
// 连续的WithName调用继续追加在后缀
// 名称段只包含字母、数字和连字符
func (l *zapLogger) WithName(name string) Logger {
	newLogger := l.zapLogger.Named(name)

	return NewLogger(newLogger, l.lumberJackLogger, l.opts)
}

// Flush 调用底层核心的Sync方法，刷新所有缓冲
// 应用程序在退出前应注意调用Sync
func Flush() { std.Flush() }

// Flush 调用底层核心的Sync方法，刷新所有缓冲
// 应用程序在退出前应注意调用Sync
func (l *zapLogger) Flush() {
	_ = l.zapLogger.Sync()
}

// NewLogger 创建一个新的Logger
func NewLogger(l *zap.Logger, logger *lumberjack.Logger, opts *Options) Logger {
	return &zapLogger{
		zapLogger: l,
		infoLogger: infoLogger{
			log:   l,
			level: zap.InfoLevel,
		},
		opts:             opts,
		lumberJackLogger: logger,
		contextKeys:      make([]string, 0),
		day:              time.Now().Day(),
	}
}

// Debug 输出Debug等级日志
func Debug(msg string, fields ...Field) {
	std.checkDay()
	std.zapLogger.Debug(msg, fields...)
}

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.checkDay()
	l.zapLogger.Debug(msg, fields...)
}

// Debugf 输出Debug等级日志，使用fmt.Sprintf格式化
func Debugf(format string, v ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Debugf(format, v...)
}

func (l *zapLogger) Debugf(format string, v ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Debugf(format, v...)
}

// Debugw 输出Debug等级日志，并携带key/value形式的值
func Debugw(msg string, keysAndValues ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Debugw(msg, keysAndValues...)
}

func (l *zapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Debugw(msg, keysAndValues...)
}

// Info 输出Info等级日志
func Info(msg string, fields ...Field) {
	std.zapLogger.Info(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.checkDay()
	l.zapLogger.Info(msg, fields...)
}

// Infof 输出Info等级日志，使用fmt.Sprintf格式化
func Infof(format string, v ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Infof(format, v...)
}

func (l *zapLogger) Infof(format string, v ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Infof(format, v...)
}

// Infow 输出Info等级日志，并携带key/value形式的值
func Infow(msg string, keysAndValues ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Infow(msg, keysAndValues...)
}

func (l *zapLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Infow(msg, keysAndValues...)
}

// Warn 输出Warn等级日志
func Warn(msg string, fields ...Field) {
	std.checkDay()
	std.zapLogger.Warn(msg, fields...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.checkDay()
	l.zapLogger.Warn(msg, fields...)
}

// Warnf 输出Debug等级日志，使用fmt.Sprintf格式化
func Warnf(format string, v ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Warnf(format, v...)
}

func (l *zapLogger) Warnf(format string, v ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Warnf(format, v...)
}

// Warnw 输出Warn等级日志，并携带key/value形式的值
func Warnw(msg string, keysAndValues ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Warnw(msg, keysAndValues...)
}

func (l *zapLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Warnw(msg, keysAndValues...)
}

// Error 输出Error等级日志
func Error(msg string, fields ...Field) {
	std.checkDay()
	std.zapLogger.Error(msg, fields...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.checkDay()
	l.zapLogger.Error(msg, fields...)
}

// Errorf 输出Error等级日志，使用fmt.Sprintf格式化
func Errorf(format string, v ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Errorf(format, v...)
}

func (l *zapLogger) Errorf(format string, v ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Errorf(format, v...)
}

// Errorw 输出Error等级日志，并携带key/value形式的值
func Errorw(msg string, keysAndValues ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Errorw(msg, keysAndValues...)
}

func (l *zapLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Errorw(msg, keysAndValues...)
}

// Panic 输出Panic等级日志
func Panic(msg string, fields ...Field) {
	std.checkDay()
	std.zapLogger.Panic(msg, fields...)
}

func (l *zapLogger) Panic(msg string, fields ...Field) {
	l.checkDay()
	l.zapLogger.Panic(msg, fields...)
}

// Panicf 输出Error等级日志，使用fmt.Sprintf格式化
func Panicf(format string, v ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Panicf(format, v...)
}

func (l *zapLogger) Panicf(format string, v ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Panicf(format, v...)
}

// Panicw 输出Panic等级日志，并携带key/value形式的值
func Panicw(msg string, keysAndValues ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Panicw(msg, keysAndValues...)
}

func (l *zapLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Panicw(msg, keysAndValues...)
}

// Fatal 输出Fatal等级日志
func Fatal(msg string, fields ...Field) {
	std.checkDay()
	std.zapLogger.Fatal(msg, fields...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.checkDay()
	l.zapLogger.Fatal(msg, fields...)
}

// Fatalf 输出Fatal等级日志，使用fmt.Sprintf格式化
func Fatalf(format string, v ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Fatalf(format, v...)
}

func (l *zapLogger) Fatalf(format string, v ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Fatalf(format, v...)
}

// Fatalw 输出Fatal等级日志，并携带key/value形式的值
func Fatalw(msg string, keysAndValues ...interface{}) {
	std.checkDay()
	std.zapLogger.Sugar().Fatalw(msg, keysAndValues...)
}

func (l *zapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.checkDay()
	l.zapLogger.Sugar().Fatalw(msg, keysAndValues...)
}

//AddContextKeys 添加多个上下文key
func AddContextKeys(keys ...string) {
	std.AddContextKeys(keys...)
}

// AddContextKeys 添加多个上下文key
func (l *zapLogger) AddContextKeys(keys ...string) {
	l.contextKeys = append(l.contextKeys, keys...)
}

//DelContextKey 删除一个上下文key
func DelContextKey(key string) {
	std.DelContextKey(key)
}

// DelContextKey 删除一个上下文key
func (l *zapLogger) DelContextKey(key string) {
	for i, k := range l.contextKeys {
		if k == key {
			l.contextKeys = append(l.contextKeys[:i], l.contextKeys[i+1:]...)
			break
		}
	}
}

// L 带有上下文的输出
func L(ctx context.Context) *zapLogger { return std.L(ctx) }

func (l *zapLogger) L(ctx context.Context) *zapLogger {
	lg := l.clone()

	for _, key := range l.contextKeys {
		if value := ctx.Value(key); value != nil {
			lg.zapLogger = lg.zapLogger.With(zap.Any(key, value))
		}
	}

	return lg
}

//go:noinline
func (l *zapLogger) clone() *zapLogger {
	copyPtr := *l
	return &copyPtr
}

// checkDay 判断是否是新的一天,如果是新的一天,则重新创建一个新的日志文件
func (l *zapLogger) checkDay() {
	now := time.Now()
	if now.Day() != l.day {
		l.day = now.Day()
		l.rotate()
	}
}

// rotate 重新创建一个新的日志文件
func (l *zapLogger) rotate() {
	mu.Lock()
	defer mu.Unlock()
	l.lumberJackLogger.Filename = l.opts.fileName()
	err := l.lumberJackLogger.Rotate()
	if err != nil {
		Errorw("Rotate log file failed", "err", err)
	}
}
