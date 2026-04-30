// Package logger installs a process-wide slog handler from a config.Logger.
package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/velociti/cdp-go/internal/config"
)

// Setup installs a JSON slog handler on stderr at the configured level
// as the process default. Unrecognized levels fall back to info.
func Setup(cfg config.Logger) {
	var lvl slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
}
