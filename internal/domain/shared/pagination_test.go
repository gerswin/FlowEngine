package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewListQuery_NormalCase(t *testing.T) {
	q := NewListQuery(2, 10)

	assert.Equal(t, 10, q.Offset) // (2-1) * 10
	assert.Equal(t, 10, q.Limit)
}

func TestNewListQuery_PageOne(t *testing.T) {
	q := NewListQuery(1, 20)

	assert.Equal(t, 0, q.Offset)
	assert.Equal(t, 20, q.Limit)
}

func TestNewListQuery_NegativePage(t *testing.T) {
	q := NewListQuery(-1, 10)

	assert.Equal(t, 0, q.Offset) // clamped to page 1
	assert.Equal(t, 10, q.Limit)
}

func TestNewListQuery_ZeroPage(t *testing.T) {
	q := NewListQuery(0, 10)

	assert.Equal(t, 0, q.Offset) // clamped to page 1
	assert.Equal(t, 10, q.Limit)
}

func TestNewListQuery_ZeroPageSize(t *testing.T) {
	q := NewListQuery(1, 0)

	assert.Equal(t, 0, q.Offset)
	assert.Equal(t, 20, q.Limit) // default 20
}

func TestNewListQuery_NegativePageSize(t *testing.T) {
	q := NewListQuery(1, -5)

	assert.Equal(t, 0, q.Offset)
	assert.Equal(t, 20, q.Limit) // default 20
}

func TestNewListQuery_OversizedPage(t *testing.T) {
	q := NewListQuery(1, 200)

	assert.Equal(t, 0, q.Offset)
	assert.Equal(t, 100, q.Limit) // capped at 100
}

func TestNewListQuery_ExactlyMaxPageSize(t *testing.T) {
	q := NewListQuery(1, 100)

	assert.Equal(t, 0, q.Offset)
	assert.Equal(t, 100, q.Limit) // exactly at cap
}

func TestNewListQuery_LargePage(t *testing.T) {
	q := NewListQuery(5, 25)

	assert.Equal(t, 100, q.Offset) // (5-1) * 25
	assert.Equal(t, 25, q.Limit)
}
