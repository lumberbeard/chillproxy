package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ChillFormatter is a custom handler that formats logs like chillstreams
type ChillFormatter struct {
	w      io.Writer
	mu     sync.Mutex
	level  slog.Leveler
	groups []string
	attrs  []slog.Attr
}

func NewChillFormatter(w io.Writer, level slog.Leveler) *ChillFormatter {
	return &ChillFormatter{
		w:     w,
		level: level,
	}
}

func (h *ChillFormatter) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *ChillFormatter) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var buf strings.Builder

	// Get scope and determine icon
	scope := ""
	var extraAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "scope" {
			scope = a.Value.String()
		} else {
			extraAttrs = append(extraAttrs, a)
		}
		return true
	})

	// Determine emoji and category based on scope or message content
	emoji, category := getEmojiAndCategory(scope, r.Message)

	// Icon
	buf.WriteString(emoji)
	buf.WriteString(" | ")

	// Level with color
	levelStr, levelColor := getLevelString(r.Level)
	buf.WriteString(levelColor)
	buf.WriteString(fmt.Sprintf("%-5s", levelStr))
	buf.WriteString("\033[0m") // Reset color
	buf.WriteString(" | ")

	// Timestamp (UTC)
	buf.WriteString(r.Time.UTC().Format("2006-01-02 15:04:05.000 MST"))
	buf.WriteString(" | ")

	// Category
	buf.WriteString(emoji)
	buf.WriteString("  ")
	buf.WriteString(fmt.Sprintf("%-8s", category))
	buf.WriteString(" | ")

	// Message
	buf.WriteString(r.Message)

	// Add extra attributes
	if len(extraAttrs) > 0 {
		for _, attr := range extraAttrs {
			if shouldSkipAttr(attr.Key) {
				continue
			}
			buf.WriteString(" ")
			buf.WriteString(attr.Key)
			buf.WriteString("=")
			buf.WriteString(formatValue(attr.Value))
		}
	}

	buf.WriteString("\n")

	_, err := h.w.Write([]byte(buf.String()))
	return err
}

func (h *ChillFormatter) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &ChillFormatter{
		w:      h.w,
		level:  h.level,
		groups: h.groups,
		attrs:  newAttrs,
	}
}

func (h *ChillFormatter) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &ChillFormatter{
		w:      h.w,
		level:  h.level,
		groups: newGroups,
		attrs:  h.attrs,
	}
}

// getEmojiAndCategory returns the appropriate emoji and category for a log entry
func getEmojiAndCategory(scope, message string) (string, string) {
	// Check scope first
	scopeLower := strings.ToLower(scope)

	// Specific scope mappings
	if strings.Contains(scopeLower, "server") || strings.Contains(scopeLower, "http") {
		return "🌐", "SERVER"
	}
	if strings.Contains(scopeLower, "database") || strings.Contains(scopeLower, "db") {
		return "🗃️", "DATABASE"
	}
	if strings.Contains(scopeLower, "cache") {
		return "🗄️", "CACHE"
	}
	if strings.Contains(scopeLower, "worker") {
		return "⚙️", "WORKER"
	}
	if strings.Contains(scopeLower, "peer") || strings.Contains(message, "peer") {
		return "🔗", "PEER"
	}
	if strings.Contains(scopeLower, "prowlarr") || strings.Contains(message, "[PROWLARR]") {
		return "🔍", "INDEXER"
	}
	if strings.Contains(scopeLower, "torz") || strings.Contains(scopeLower, "stremio") {
		return "🎬", "STREMIO"
	}
	if strings.Contains(scopeLower, "torbox") || strings.Contains(message, "TORBOX") {
		return "📦", "TORBOX"
	}
	if strings.Contains(scopeLower, "auth") || strings.Contains(message, "auth") {
		return "🔑", "AUTH"
	}
	if strings.Contains(scopeLower, "pool") || strings.Contains(message, "pool") {
		return "💛", "POOL"
	}

	// Default
	return "📝", "APP"
}

// getLevelString returns the level string and color code
func getLevelString(level slog.Level) (string, string) {
	switch {
	case level < slog.LevelDebug: // TRACE
		return "TRACE", "\033[35m" // Magenta
	case level < slog.LevelInfo: // DEBUG
		return "DEBUG", "\033[35m" // Magenta
	case level < slog.LevelWarn: // INFO
		return "INFO", "\033[32m" // Green
	case level < slog.LevelError: // WARN
		return "WARN", "\033[33m" // Yellow
	default: // ERROR, FATAL
		return "ERROR", "\033[31m" // Red
	}
}

// shouldSkipAttr determines if an attribute should be skipped
func shouldSkipAttr(key string) bool {
	// Skip request IDs that are too verbose
	if key == "req.id" {
		return true
	}
	// Skip internal fields
	if strings.HasPrefix(key, "_") {
		return true
	}
	return false
}

// formatValue formats an attribute value for display
func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		// Truncate very long values
		if len(s) > 100 {
			return s[:97] + "..."
		}
		return s
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	default:
		return fmt.Sprint(v.Any())
	}
}

// Helper function to get caller info (file:line)
func getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	// Get just the filename without path
	idx := strings.LastIndexByte(file, '/')
	if idx >= 0 {
		file = file[idx+1:]
	}
	return fmt.Sprintf("%s:%d", file, line)
}
