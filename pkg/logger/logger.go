// Package logger 基于 Zap 的日志封装，提供全局 L 变量
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger

// Init 初始化 Logger
func Init(level, logFile string) {
	cfg := zap.NewProductionConfig()
	// 设置日志级别
	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.OutputPaths = []string{logFile, "stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	L, _ = cfg.Build()
	zap.ReplaceGlobals(L)
}
