package cache

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryCacheGetSetDelete(t *testing.T) {
	c := NewMemoryCache()

	if _, ok := c.Get("missing"); ok {
		t.Fatal("empty cache reported a hit")
	}

	c.Set("k", []byte("v"), 0)

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected hit")
	}
	if string(got) != "v" {
		t.Fatalf("value = %q, want v", got)
	}
	if !c.Has("k") {
		t.Fatal("Has = false, want true")
	}

	c.Delete("k")

	if c.Has("k") {
		t.Fatal("Has = true after delete")
	}

	// Deleting twice must be a no-op, not a panic.
	c.Delete("k")
}

func TestMemoryCacheNegativeTTLDeletes(t *testing.T) {
	c := NewMemoryCache()
	c.Set("k", []byte("v"), 0)
	c.Set("k", []byte("v2"), -time.Second)

	if c.Has("k") {
		t.Fatal("negative ttl stored the value")
	}
}

func TestMemoryCacheExpires(t *testing.T) {
	c := NewMemoryCache()
	c.Set("k", []byte("v"), 20*time.Millisecond)

	time.Sleep(40 * time.Millisecond)

	if c.Has("k") {
		t.Fatal("entry survived its ttl")
	}
}

func TestMemoryCacheCopiesValues(t *testing.T) {
	c := NewMemoryCache()

	src := []byte("abc")
	c.Set("k", src, 0)

	// Mutating the source must not change the cached bytes.
	src[0] = 'z'

	got, _ := c.Get("k")
	if string(got) != "abc" {
		t.Fatalf("cached value changed to %q", got)
	}

	// Mutating the returned slice must not change the cache either.
	got[0] = 'y'

	again, _ := c.Get("k")
	if string(again) != "abc" {
		t.Fatalf("cache leaked its internal slice: %q", again)
	}
}

func TestMemoryCacheSetNX(t *testing.T) {
	c := NewMemoryCache()

	stored, err := c.SetNX("k", []byte("1"), 0)
	if err != nil || !stored {
		t.Fatalf("first SetNX = %v, %v; want true, nil", stored, err)
	}

	stored, err = c.SetNX("k", []byte("2"), 0)
	if err != nil || stored {
		t.Fatalf("second SetNX = %v, %v; want false, nil", stored, err)
	}
}

func TestMemoryCacheSetNXExpiredReclaim(t *testing.T) {
	c := NewMemoryCache()
	c.SetNX("k", []byte("1"), 10*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	stored, err := c.SetNX("k", []byte("2"), 0)
	if err != nil || !stored {
		t.Fatalf("expired key not reclaimable: %v, %v", stored, err)
	}
}

func TestMemoryCacheSetNXConcurrent(t *testing.T) {
	c := NewMemoryCache()

	const workers = 64

	var wins atomic.Int64

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()

			stored, err := c.SetNX("race", []byte(strconv.Itoa(i)), 0)
			if err != nil {
				t.Error(err)
				return
			}
			if stored {
				wins.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if wins.Load() != 1 {
		t.Fatalf("SetNX winners = %d, want exactly 1", wins.Load())
	}
}

func BenchmarkMemoryCacheGet(b *testing.B) {
	c := NewMemoryCache()
	c.Set("k", []byte("v"), 0)

	for i := 0; i < b.N; i++ {
		_, _ = c.Get("k")
	}
}
