package plates_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/gofoji/plates"
)

func TestMap(t *testing.T) {
	c := plates.NewMap()

	if _, ok := c.Get("missing"); ok {
		t.Errorf("empty Map returned a value")
	}

	a := plates.NewEngine(plates.FormatEngine{})
	b := plates.NewEngine(plates.FormatEngine{})

	c.Set("a", a)
	c.Set("b", b)

	got, ok := c.Get("a")
	if !ok {
		t.Errorf("`a` not found")
	}

	if got != a {
		t.Errorf("got = %v, want %v", got, a)
	}

	c.Set("a", b)

	if got, _ = c.Get("a"); got != b {
		t.Errorf("Set did not overwrite: got = %v, want %v", got, b)
	}

	c.Delete("a")

	if _, ok = c.Get("a"); ok {
		t.Errorf("`a` found after Delete")
	}

	if _, ok = c.Get("b"); !ok {
		t.Errorf("Delete removed the wrong key")
	}

	c.Delete("missing") // no-op

	c.Clear()

	if _, ok = c.Get("b"); ok {
		t.Errorf("`b` found after Clear")
	}

	// Usable again after Clear.
	c.Set("c", a)

	if _, ok = c.Get("c"); !ok {
		t.Errorf("`c` not found after Clear")
	}
}

// TestMapZeroValue verifies the zero value is usable without NewMap.
func TestMapZeroValue(t *testing.T) {
	var c plates.Map

	if _, ok := c.Get("missing"); ok {
		t.Errorf("zero Map returned a value")
	}

	c.Delete("missing")
	c.Clear()

	e := plates.NewEngine(plates.FormatEngine{})
	c.Set("a", e)

	if got, ok := c.Get("a"); !ok || got != e {
		t.Errorf("got = %v, %v, want %v, true", got, ok, e)
	}
}

// TestMapSnapshotIsolation verifies a reader holding a value is unaffected by concurrent writes, which is the
// property copy-on-write buys over mutating a shared map.
func TestMapSnapshotIsolation(t *testing.T) {
	c := plates.NewMap()
	e := plates.NewEngine(plates.FormatEngine{})
	c.Set("a", e)

	got, _ := c.Get("a")

	c.Set("a", plates.NewEngine(plates.FormatEngine{}))
	c.Delete("a")
	c.Clear()

	if got != e {
		t.Errorf("read value changed under concurrent writes")
	}
}

func BenchmarkMapGetParallel(b *testing.B) {
	c := plates.NewMap()
	e := plates.NewEngine(plates.FormatEngine{})

	for i := range 32 {
		c.Set(strconv.Itoa(i), e)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(strconv.Itoa(i % 32))
			i++
		}
	})
}

// TestMapConcurrent exercises the lockless read path against concurrent writers. Meaningful under -race.
func TestMapConcurrent(t *testing.T) {
	const (
		keys    = 32
		readers = 8
		writers = 4
		rounds  = 200
	)

	c := plates.NewMap()
	e := plates.NewEngine(plates.FormatEngine{})

	for i := range keys {
		c.Set(strconv.Itoa(i), e)
	}

	var wg sync.WaitGroup

	for range readers {
		wg.Go(func() {
			for i := range rounds * keys {
				if v, ok := c.Get(strconv.Itoa(i % keys)); ok && v != e {
					t.Errorf("torn read: %v", v)
					return
				}
			}
		})
	}

	for w := range writers {
		wg.Go(func() {
			for i := range rounds {
				k := strconv.Itoa((i + w) % keys)
				c.Set(k, e)

				if i%8 == 0 {
					c.Delete(k)
				}

				if i%64 == 0 {
					c.Clear()
				}
			}
		})
	}

	wg.Wait()
}
