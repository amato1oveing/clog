package clog

import (
	"go.uber.org/zap/zapcore"
)

// 定义常用上下文携带字段
const (
	KeyRequestID   string = "request_id"
	KeyUsername    string = "username"
	KeyUserId      string = "user_id"
	KeyWatcherName string = "watcher"
	KeyTraceId     string = "trace_id"
)

// Field 是一个zapcore.Field别名
type Field = zapcore.Field

// Level 是一个zapcore.Level别名
type Level = zapcore.Level

// 定义日志等级
var (
	DebugLevel  Level = zapcore.DebugLevel
	InfoLevel   Level = zapcore.InfoLevel
	WarnLevel   Level = zapcore.WarnLevel
	ErrorLevel  Level = zapcore.ErrorLevel
	DPanicLevel Level = zapcore.DPanicLevel
	PanicLevel  Level = zapcore.PanicLevel
	FatalLevel  Level = zapcore.FatalLevel
)

// 定义常用的时间格式化样式
var (
	TimeFormat = "2006-01-02"
)
