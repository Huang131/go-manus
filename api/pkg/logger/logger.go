package logger

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 日志级别名称到 zapcore.Level 的映射
var levelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

var (
	// encoder 配置（创建后不可变）
	encoderConfig zapcore.EncoderConfig
	// encoder 实例
	encoder zapcore.Encoder
	// writeSyncer 输出目标
	writeSyncer zapcore.WriteSyncer
	// 原子级别，支持动态调整
	atomicLevel zap.AtomicLevel
	// 日志实例
	log *zap.Logger
	// 互斥锁
	mu sync.RWMutex
	// 是否已初始化
	inited bool
)

// Config 日志配置
type Config struct {
	Level      string // 日志级别: debug, info, warn, error
	Filename   string // 日志文件路径，为空则只输出到 stdout
	MaxSize    int    // 单个日志文件最大大小(MB)
	MaxBackups int    // 保留的旧日志文件数量
	MaxAge     int    // 旧日志文件保留天数
	Compress   bool   // 是否压缩旧日志
}

// Init 初始化日志系统
// level 支持: debug, info, warn, error（默认 info）
func Init(level string) error {
	return InitWithConfig(Config{Level: level})
}

// InitWithConfig 使用配置初始化日志系统
func InitWithConfig(cfg Config) error {
	// 解析日志级别
	level := cfg.Level
	if level == "" {
		level = "info"
	}
	zapLevel, ok := levelMap[level]
	if !ok {
		zapLevel = zapcore.InfoLevel
	}

	mu.Lock()
	defer mu.Unlock()

	// 初始化原子级别（支持动态调整）
	atomicLevel = zap.NewAtomicLevelAt(zapLevel)

	// 初始化编码器配置
	encoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	encoder = zapcore.NewJSONEncoder(encoderConfig)

	// 确定输出位置
	if cfg.Filename != "" {
		// 使用 lumberjack 实现日志切割
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		// 同时输出到文件和控制台
		writeSyncer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			zapcore.AddSync(lumberjackLogger),
		)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建 Core
	core := zapcore.NewCore(encoder, writeSyncer, atomicLevel)
	// AddCaller: true 启用调用方信息
	// AddCallerSkip(2) 跳过:
	// 1. zap 内部的 write/Check 调用
	// 2. log.Info() <- zap.Logger 的方法
	// 这样 caller 会显示业务调用方
	log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	inited = true

	return nil
}

// Get 返回全局日志实例
func Get() *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if !inited {
		// 兜底初始化
		log, _ = zap.NewProduction()
	}
	return log
}

// SetLevel 动态调整日志级别
// level 支持: debug, info, warn, error
func SetLevel(level string) {
	zapLevel, ok := levelMap[level]
	if !ok {
		zapLevel = zapcore.InfoLevel
	}

	mu.RLock()
	atomicLevel.SetLevel(zapLevel)
	mu.RUnlock()
}

// GetLevel 获取当前日志级别
func GetLevel() string {
	mu.RLock()
	defer mu.RUnlock()

	currentLevel := atomicLevel.Level()
	for name, lvl := range levelMap {
		if lvl == currentLevel {
			return name
		}
	}
	return "info"
}

// Sync 刷新日志缓冲区
func Sync() {
	mu.RLock()
	defer mu.RUnlock()
	if log != nil {
		_ = log.Sync()
	}
}

// ==================== 日志方法封装 ====================

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal 致命日志
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// ==================== Field 构造函数（隐藏 zap 依赖） ====================

// String 添加字符串字段
func String(key string, val string) zap.Field {
	return zap.String(key, val)
}

// Int 添加整数字段
func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

// Int64 添加 int64 字段
func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

// Float64 添加 float64 字段
func Float64(key string, val float64) zap.Field {
	return zap.Float64(key, val)
}

// Bool 添加布尔字段
func Bool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

// Err 添加错误字段
func Err(err error) zap.Field {
	return zap.Error(err)
}

// Any 添加任意类型字段
func Any(key string, val any) zap.Field {
	return zap.Any(key, val)
}

// Strings 添加字符串数组字段
func Strings(key string, val []string) zap.Field {
	return zap.Strings(key, val)
}

// Dur 添加 Duration 字段
func Dur(key string, val time.Duration) zap.Field {
	return zap.Duration(key, val)
}

// Object 添加对象字段（实现ObjectMarshaler接口）
func Object(key string, val zapcore.ObjectMarshaler) zap.Field {
	return zap.Object(key, val)
}
