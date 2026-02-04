package logging

import (
	"io"
	"log/slog"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Component constants for structured logging.
const (
	CompStatus  = "status"
	CompMCP     = "mcp"
	CompNotif   = "notif"
	CompPerf    = "perf"
	CompUI      = "ui"
	CompSession = "session"
	CompStorage = "storage"
	CompPool    = "pool"
	CompHTTP    = "http"
)

// Config holds logging configuration.
type Config struct {
	// LogDir is the directory for log files (e.g. ~/.agent-deck)
	LogDir string

	// Level is the minimum log level: "debug", "info", "warn", "error"
	Level string

	// Format is "json" (default) or "text"
	Format string

	// MaxSizeMB is the max size in MB before rotation (default: 10)
	MaxSizeMB int

	// MaxBackups is rotated files to keep (default: 5)
	MaxBackups int

	// MaxAgeDays is days to keep rotated files (default: 10)
	MaxAgeDays int

	// Compress rotated files (default: true)
	Compress bool

	// RingBufferSize is the in-memory ring buffer size in bytes (default: 10MB)
	RingBufferSize int

	// AggregateIntervalSecs is the aggregation flush interval (default: 30)
	AggregateIntervalSecs int

	// PprofEnabled starts pprof server on localhost:6060
	PprofEnabled bool

	// Debug indicates whether debug mode is active
	Debug bool
}

var (
	globalLogger *slog.Logger
	globalRing   *RingBuffer
	globalAgg    *Aggregator
	globalMu     sync.RWMutex
	lumberjackW  *lumberjack.Logger
)

// Init initializes the global logging system.
// When debug is false and no log dir is provided, logs are discarded.
func Init(cfg Config) {
	globalMu.Lock()
	defer globalMu.Unlock()

	// Defaults
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 10
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 5
	}
	if cfg.MaxAgeDays <= 0 {
		cfg.MaxAgeDays = 10
	}
	if cfg.RingBufferSize <= 0 {
		cfg.RingBufferSize = 10 * 1024 * 1024 // 10MB
	}
	if cfg.AggregateIntervalSecs <= 0 {
		cfg.AggregateIntervalSecs = 30
	}

	// Parse level
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	// If not in debug mode and no explicit log dir, discard everything
	if !cfg.Debug && cfg.LogDir == "" {
		globalLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
		globalRing = NewRingBuffer(1024) // minimal
		globalAgg = NewAggregator(nil, cfg.AggregateIntervalSecs)
		return
	}

	// Set up lumberjack for rotation
	logPath := filepath.Join(cfg.LogDir, "debug.log")
	lumberjackW = &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}

	// Ring buffer for crash dumps
	globalRing = NewRingBuffer(cfg.RingBufferSize)

	// MultiWriter: lumberjack + ring buffer
	multi := io.MultiWriter(lumberjackW, globalRing)

	// Create handler
	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(multi, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(multi, handlerOpts)
	}

	globalLogger = slog.New(handler)

	// Aggregator
	globalAgg = NewAggregator(globalLogger, cfg.AggregateIntervalSecs)
	globalAgg.Start()

	// pprof
	if cfg.PprofEnabled {
		startPprof()
	}
}

// Logger returns the global logger. Safe to call before Init (returns default).
func Logger() *slog.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalLogger == nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return globalLogger
}

// ForComponent returns a sub-logger with the component field set.
func ForComponent(name string) *slog.Logger {
	return Logger().With(slog.String("component", name))
}

// Aggregate records a high-frequency event for batched logging.
func Aggregate(component, key string, fields ...slog.Attr) {
	globalMu.RLock()
	agg := globalAgg
	globalMu.RUnlock()
	if agg != nil {
		agg.Record(component, key, fields...)
	}
}

// DumpRingBuffer writes the ring buffer contents to a file.
func DumpRingBuffer(path string) error {
	globalMu.RLock()
	ring := globalRing
	globalMu.RUnlock()
	if ring == nil {
		return nil
	}
	return ring.DumpToFile(path)
}

// Shutdown flushes the aggregator and closes writers.
func Shutdown() {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalAgg != nil {
		globalAgg.Stop()
		globalAgg = nil
	}
	if lumberjackW != nil {
		lumberjackW.Close()
		lumberjackW = nil
	}
	globalLogger = nil
	globalRing = nil
}
