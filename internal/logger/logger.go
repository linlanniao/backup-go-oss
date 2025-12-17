package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

var (
	// Logger 全局日志实例
	Logger *slog.Logger
)

// multiHandler 将日志同时写入多个 handler
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m.handlers {
		if err := h.Handle(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// InitLogger 初始化日志器
// level: 日志级别，可选值: "debug", "info", "warn", "error"
// logDir: 日志目录，如果指定则会将日志输出到文件，否则输出到标准错误
func InitLogger(level string, logDir string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// 创建终端 handler（带颜色）
	terminalHandler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.DateTime,
		NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
		AddSource:  true,
	})

	var handlers []slog.Handler
	handlers = append(handlers, terminalHandler)

	if logDir != "" {
		// 确保日志目录存在
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "创建日志目录失败: %v\n", err)
		} else {
			// 创建日志文件，文件名包含时间戳
			timestamp := time.Now().Format("20060102-150405")
			logFile := filepath.Join(logDir, fmt.Sprintf("backup-%s.log", timestamp))
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "打开日志文件失败: %v\n", err)
			} else {
				// 创建文件 handler（纯文本，无颜色）
				// 使用 NewTextHandler 确保输出纯文本格式，不包含任何 ANSI 转义码
				fileHandler := slog.NewTextHandler(file, &slog.HandlerOptions{
					Level:     logLevel,
					AddSource: true,
				})
				handlers = append(handlers, fileHandler)
				// 输出日志文件路径信息（只输出到终端，不写入文件）
				fmt.Fprintf(os.Stderr, "日志文件已创建: %s\n", logFile)
			}
		}
	}

	// 如果只有一个 handler，直接使用；否则使用 multiHandler
	if len(handlers) == 1 {
		Logger = slog.New(handlers[0])
	} else {
		Logger = slog.New(&multiHandler{handlers: handlers})
	}
}

// InitDefaultLogger 使用默认配置初始化日志器
func InitDefaultLogger() {
	InitLogger("info", "")
}

// Debug 记录调试日志
func Debug(msg string, args ...any) {
	if Logger != nil {
		Logger.Debug(msg, args...)
	}
}

// Info 记录信息日志
func Info(msg string, args ...any) {
	if Logger != nil {
		Logger.Info(msg, args...)
	}
}

// Warn 记录警告日志
func Warn(msg string, args ...any) {
	if Logger != nil {
		Logger.Warn(msg, args...)
	}
}

// Error 记录错误日志
func Error(msg string, args ...any) {
	if Logger != nil {
		Logger.Error(msg, args...)
	}
}
