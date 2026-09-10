package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// contextKey 使用私有类型，避免不同包的 context key 发生碰撞。
type contextKey struct{}

var requestIDKey contextKey

const requestIDField = "request_id"

// WithRequestID 将请求 ID 写入上下文，供请求链路中的日志使用。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext 从上下文读取请求 ID。
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

// 日志级别常量。对齐 config.LoggerConfig 的 oneof tag，不可直接用于 struct tag。
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// levelMap 把字符串级别映射到 zapcore.Level。
var levelMap = map[string]zapcore.Level{
	LevelDebug: zapcore.DebugLevel,
	LevelInfo:  zapcore.InfoLevel,
	LevelWarn:  zapcore.WarnLevel,
	LevelError: zapcore.ErrorLevel,
}

var (
	// encoder 实例
	encoder zapcore.Encoder
	// writeSyncer 输出目标
	writeSyncer zapcore.WriteSyncer

	// 原子级别，SetLevel/GetLevel 线程安全，无需全局锁
	atomicLevel zap.AtomicLevel
	// 日志实例（使用原子操作，Get/Sync 无需锁）
	log atomic.Pointer[zap.Logger]
	// 原子级别写入保护（在 InitWithConfig 中使用写锁）
	muLevel sync.Mutex
)

// neverCloseSyncer 包装一个 WriteSyncer，将其 Close() 和 Sync() 方法变成无操作。
//
// 设计意图：保护进程级共享资源（如 os.Stdout、os.Stderr），避免被
// MultiWriteSyncer.Close() 递归关闭。同时抑制 Sync() 错误，防止 zap
// 在 Sync() 失败时尝试记录内部错误日志导致无限递归。
//
// 为什么也要覆盖 Sync()？
// os.Stdout.Sync() 在 fd 已关闭时返回 EBADF。zap 的 Sync() 失败后会调用
// internalDriverError 记录到自己的内部 logger，内部 logger 又尝试写 stdout，
// 再触发 Sync() 失败 → 无限递归。用 neverCloseSyncer 包装后，Sync() 的
// 错误被吞掉（stdout 本来就无缓冲可刷，fd 关闭后无意义），从而切断递归链。
//
// 这种模式也适用于其他场景：
//   - 数据库连接池（连接由连接池管理， Close() 只归还不关闭 fd）
//   - 共享的 HTTP 客户端（Close() 不关闭底层 TCP 连接）
//   - 包装既有资源的 ReadCloser/WriteCloser（如 json.Decoder 包装
//     os.Stdin，不应关闭 stdin）
type neverCloseSyncer struct {
	zapcore.WriteSyncer
}

func (n neverCloseSyncer) Close() error { return nil }

// Sync 总是返回 nil，抑制底层 syncer 的同步错误。
// 理由：os.Stdout 是无缓冲的终端 fd，关闭后无同步意义，且 Sync 错误会触发
// zap 内部错误日志的无限递归（见 above）。对于日志优雅关闭场景，
// os.Stdout 的最后几条日志丢失可接受。
func (n neverCloseSyncer) Sync() error { return nil }

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
		level = LevelInfo
	}
	zapLevel, ok := levelMap[level]
	if !ok {
		zapLevel = zapcore.InfoLevel
	}

	muLevel.Lock()
	defer muLevel.Unlock()

	// 在覆盖前释放旧实例：触发最后一次 Flush，并把旧的 WriteSyncer 关闭（若实现 Closer）。
	// 这样热重载日志时不会泄露 lumberjack 的 fd 或 stdout buffer。
	shutdownCurrent()

	// 初始化原子级别（支持动态调整）
	atomicLevel = zap.NewAtomicLevelAt(zapLevel)

	// 初始化编码器
	encoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
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
	})

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
		// 同时输出到文件和控制台。
		// 用 neverCloseSyncer 包装 os.Stdout，确保 MultiWriteSyncer.Close()
		// 递归关闭子 syncer 时不会关闭 stdout fd，且 Sync() 错误被抑制。
		writeSyncer = zapcore.NewMultiWriteSyncer(
			neverCloseSyncer{zapcore.AddSync(os.Stdout)},
			zapcore.AddSync(lumberjackLogger),
		)
	} else {
		// 纯 stdout 模式：也必须用 neverCloseSyncer 包装 os.Stdout。
		// 否则 shutdownCurrent 中的 Close() 会直接关闭 os.Stdout 的 fd，
		// 导致后续所有 stdout 写操作失败（fd 已关闭，EBADF）。
		writeSyncer = neverCloseSyncer{zapcore.AddSync(os.Stdout)}
	}

	// 创建 Core
	core := zapcore.NewCore(encoder, writeSyncer, atomicLevel)
	// AddCaller: true 启用调用方信息
	// AddCallerSkip(2) 跳过:
	// 1. zap 内部的 write/Check 调用
	// 2. log.Info() <- zap.Logger 的方法
	// 这样 caller 会显示业务调用方
	log.Store(zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2)))

	return nil
}

// Get 返回全局日志实例
// 无需锁：atomic.Pointer.Load() 原子操作线程安全
func Get() *zap.Logger {
	if l := log.Load(); l != nil {
		return l
	}
	// 兜底：不应发生（InitWithConfig 失败），防御性处理
	tmp, _ := zap.NewProduction()
	return tmp
}

// WithContext 返回携带请求上下文字段的 logger。
func WithContext(ctx context.Context) *zap.Logger {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		return Get()
	}
	return Get().With(zap.String(requestIDField, requestID))
}

// SetLevel 动态调整日志级别
// 无需锁：atomicLevel.SetLevel() 本身线程安全，levelMap 是只读 map
func SetLevel(level string) {
	zapLevel, ok := levelMap[level]
	if !ok {
		zapLevel = zapcore.InfoLevel
	}
	atomicLevel.SetLevel(zapLevel)
}

// GetLevel 获取当前日志级别
// 无需锁：atomicLevel.Level() 本身线程安全
func GetLevel() string {
	currentLevel := atomicLevel.Level()
	for name, lvl := range levelMap {
		if lvl == currentLevel {
			return name
		}
	}
	return LevelInfo
}

// Sync 刷新日志缓冲区。
//
// 返回值供调用方在退出前显式处理；典型场景是把 error 输出到 stderr，
// 避免关闭阶段丢失日志（例如 stderr 写文件但文件已轮转等场景）。
func Sync() error {
	l := log.Load()
	if l == nil {
		return nil
	}
	if err := l.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "logger sync failed: %v\n", err)
		return err
	}
	return nil
}

// shutdownCurrent 在持有写锁的前提下释放当前 logger 句柄。
//
// 职责：1) Swap 旧 logger 并 Flush；2) 关闭底层 WriteSyncer（如 lumberjack）以避免 fd 泄露。
// 使用 atomic.Pointer.Swap() 原子交换，同时获取旧值并置 nil。
//
// 注意：stdout 对应的 WriteSyncer 永远不会被关闭——os.Stdout 是进程级共享资源，
// 关闭后无法恢复。这通过 InitWithConfig 中使用 neverCloseSyncer 包装 stdout 来保证。
// 对于 file 模式，writeSyncer 通常是 MultiWriteSyncer，其 Close() 会同时关闭子 syncer，
// 但因为 stdout 已被 neverCloseSyncer 包装，Close() 调用变成无操作，不会伤及 stdout fd。
// 同时 neverCloseSyncer.Sync() 返回 nil，抑制同步错误，防止 zap 内部日志递归。
func shutdownCurrent() {
	if oldLog := log.Swap(nil); oldLog != nil {
		_ = oldLog.Sync()
	}
	if writeSyncer != nil {
		if closer, ok := writeSyncer.(io.Closer); ok {
			_ = closer.Close()
		}
	}
	writeSyncer = nil
	encoder = nil
}

// ==================== 日志方法封装 ====================

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// DebugContext 记录携带上下文的调试日志。
func DebugContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// InfoContext 记录携带上下文的普通日志。
func InfoContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// WarnContext 记录携带上下文的警告日志。
func WarnContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// ErrorContext 记录携带上下文的错误日志。
func ErrorContext(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Error(msg, fields...)
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
