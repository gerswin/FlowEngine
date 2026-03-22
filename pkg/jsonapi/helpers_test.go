package jsonapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaginate_FirstPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	opts := QueryOptions{PageNumber: 1, PageSize: 3}
	page, total := Paginate(items, opts)
	assert.Equal(t, []int{1, 2, 3}, page)
	assert.Equal(t, 10, total)
}

func TestPaginate_SecondPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	opts := QueryOptions{PageNumber: 2, PageSize: 3}
	page, total := Paginate(items, opts)
	assert.Equal(t, []int{4, 5, 6}, page)
	assert.Equal(t, 10, total)
}

func TestPaginate_BeyondTotal(t *testing.T) {
	items := []int{1, 2, 3}
	opts := QueryOptions{PageNumber: 5, PageSize: 3}
	page, total := Paginate(items, opts)
	assert.Empty(t, page)
	assert.Equal(t, 3, total)
}

func TestPaginate_PageSizeLargerThanTotal(t *testing.T) {
	items := []int{1, 2, 3}
	opts := QueryOptions{PageNumber: 1, PageSize: 100}
	page, total := Paginate(items, opts)
	assert.Equal(t, []int{1, 2, 3}, page)
	assert.Equal(t, 3, total)
}

func TestPaginate_EmptySlice(t *testing.T) {
	items := []int{}
	opts := QueryOptions{PageNumber: 1, PageSize: 10}
	page, total := Paginate(items, opts)
	assert.Empty(t, page)
	assert.Equal(t, 0, total)
}

func TestPaginate_TotalAlwaysFullCount(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	opts := QueryOptions{PageNumber: 2, PageSize: 2}
	page, total := Paginate(items, opts)
	assert.Equal(t, []string{"c", "d"}, page)
	assert.Equal(t, 5, total)
}
