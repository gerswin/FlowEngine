package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		Init()
	})
}

func TestInit_MultipleCalls(t *testing.T) {
	// sync.Once ensures this is safe
	assert.NotPanics(t, func() {
		Init()
		Init()
		Init()
	})
}

func TestGet_ReturnsNonNil(t *testing.T) {
	l := Get()
	assert.NotNil(t, l)
}

func TestInfo_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		Info("test message", "key", "value")
	})
}

func TestDebug_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		Debug("test debug", "count", 42)
	})
}

func TestWarn_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		Warn("test warn")
	})
}

func TestError_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		Error("test error", "err", "something broke")
	})
}

func TestWith_ReturnsNonNil(t *testing.T) {
	l := With("component", "scheduler")
	assert.NotNil(t, l)
}

func TestWithContext_ReturnsNonNil(t *testing.T) {
	ctx := context.Background()
	l := WithContext(ctx)
	assert.NotNil(t, l)
}
