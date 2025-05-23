package sl

import (
	"log/slog"
)

// Err creates a slog.Attr with the given error.
func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
