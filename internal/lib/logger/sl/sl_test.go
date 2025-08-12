package sl_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/UnknownOlympus/hephaestus/internal/lib/logger/sl"
	"github.com/stretchr/testify/assert"
)

func TestErr(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer // buffer for log capturing
	// Create slog.Logger, which writes in logBuf
	testLogger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{}))

	errAttr := sl.Err(assert.AnError)
	testLogger.Warn("expected result:", errAttr)

	loggedOutput := logBuf.String()

	assert.Contains(t, loggedOutput, assert.AnError.Error())
}
