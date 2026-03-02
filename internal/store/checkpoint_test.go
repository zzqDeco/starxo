package store

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryStore(t *testing.T) {
	s := NewInMemoryStore()
	require.NotNil(t, s)
}

func TestSetAndGet(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	err := s.Set(ctx, "key1", []byte("value1"))
	require.NoError(t, err)

	val, ok, err := s.Get(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("value1"), val)
}

func TestGetNonExistent(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	val, ok, err := s.Get(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestSetOverwrite(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "k", []byte("v1")))
	require.NoError(t, s.Set(ctx, "k", []byte("v2")))

	val, ok, err := s.Get(ctx, "k")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("v2"), val)
}

func TestSetEmptyValue(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "empty", []byte{}))

	val, ok, err := s.Get(ctx, "empty")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte{}, val)
}

func TestSetNilValue(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "nil", nil))

	val, ok, err := s.Get(ctx, "nil")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, val)
}

func TestMultipleKeys(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		val := []byte(fmt.Sprintf("value-%d", i))
		require.NoError(t, s.Set(ctx, key, val))
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		expected := []byte(fmt.Sprintf("value-%d", i))
		val, ok, err := s.Get(ctx, key)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, expected, val)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()
	const goroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := fmt.Sprintf("g%d-k%d", id, i)
				val := []byte(fmt.Sprintf("g%d-v%d", id, i))
				if err := s.Set(ctx, key, val); err != nil {
					t.Errorf("Set failed: %v", err)
					return
				}
				got, ok, err := s.Get(ctx, key)
				if err != nil {
					t.Errorf("Get failed: %v", err)
					return
				}
				if !ok {
					t.Errorf("key %s not found after Set", key)
					return
				}
				if string(got) != string(val) {
					t.Errorf("key %s: got %s, want %s", key, got, val)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}
