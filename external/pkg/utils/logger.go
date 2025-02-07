package utils

import (
	"log"
	"path/filepath"
	"runtime"
	"sync/atomic"
)

// Logger 定义日志接口
type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})

	Warn(v ...interface{})
	Warnf(format string, v ...interface{})

	Error(v ...interface{})
	Errorf(format string, v ...interface{})
}

var globalLogger atomic.Value

func init() {
	SetLogger(newDefaultLogger()) // 初始化默认日志
}

// SetLogger 设置全局日志实例
func SetLogger(l Logger) {
	globalLogger.Store(l)
}

// GetLogger 获取全局日志实例
func GetLogger() Logger {
	return globalLogger.Load().(Logger)
}

// defaultLogger 默认日志实现
type defaultLogger struct {
	enableCallerInfo atomic.Bool // 是否启用调用者信息
}

// NewDefaultLogger 创建一个默认日志实例
func newDefaultLogger() *defaultLogger {
	return &defaultLogger{}
}

// EnableCallerInfo 动态控制是否启用文件和行号打印
func (d *defaultLogger) EnableCallerInfo(enable bool) {
	d.enableCallerInfo.Store(enable)
}

// getCallerInfo 获取调用栈的文件名和行号
func (d *defaultLogger) getCallerInfo() (file string, line int) {
	// 参数3表示跳过三层调用栈（getCallerInfo、Debug/Debugf、实际调用者）
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "unknown"
		line = 0
	}
	// 只保留文件名，去掉路径
	file = filepath.Base(file)
	return file, line
}

// logWithCallerInfo 打印带调用者信息的日志
func (d *defaultLogger) logWithCallerInfo(level, format string, v ...interface{}) {
	if d.enableCallerInfo.Load() {
		file, line := d.getCallerInfo()
		log.Printf("[%s] %s:%d - "+format, append([]interface{}{level, file, line}, v...)...)
	} else {
		log.Printf("[%s] "+format, append([]interface{}{level}, v...)...)
	}
}

// Debug 打印调试日志
func (d *defaultLogger) Debug(v ...interface{}) {
	d.logWithCallerInfo("DEBUG", "%s", v...)
}

// Debugf 格式化打印调试日志
func (d *defaultLogger) Debugf(format string, v ...interface{}) {
	d.logWithCallerInfo("DEBUG", format, v...)
}

// Info 打印信息日志
func (d *defaultLogger) Info(v ...interface{}) {
	d.logWithCallerInfo("INFO", "%s", v...)
}

// Infof 格式化打印信息日志
func (d *defaultLogger) Infof(format string, v ...interface{}) {
	d.logWithCallerInfo("INFO", format, v...)
}

// Warn 打印警告日志
func (d *defaultLogger) Warn(v ...interface{}) {
	d.logWithCallerInfo("WARN", "%s", v...)
}

// Warnf 格式化打印警告日志
func (d *defaultLogger) Warnf(format string, v ...interface{}) {
	d.logWithCallerInfo("WARN", format, v...)
}

// Error 打印错误日志
func (d *defaultLogger) Error(v ...interface{}) {
	d.logWithCallerInfo("ERROR", "%s", v...)
}

// Errorf 格式化打印错误日志
func (d *defaultLogger) Errorf(format string, v ...interface{}) {
	d.logWithCallerInfo("ERROR", format, v...)
}
