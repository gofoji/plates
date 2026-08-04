package plates

import (
	"maps"
	"sync"
	"sync/atomic"
)

// Map is a thread-safe cache for templates, tuned for the read-mostly access pattern a template cache actually has:
// a bounded key set, written once at load and read on every render.
//
// Reads are lockless — a single atomic load — so readers never contend with each other or with a writer.  Writes
// copy the map under a mutex that serializes writers only, making a write O(n) in the number of cached templates.
// That is the intended trade; if a workload writes as often as it reads, this is the wrong container for it.
//
// The zero value is ready to use.
type Map struct {
	mu    sync.Mutex
	items atomic.Pointer[map[string]Template]
}

func NewMap() *Map {
	return &Map{}
}

// load returns the current items, which may be nil.  A nil map reads as empty, so callers need not check.
func (c *Map) load() map[string]Template {
	if m := c.items.Load(); m != nil {
		return *m
	}

	return nil
}

// clone copies the current items so they can be mutated without disturbing in-flight readers.
// The caller must hold c.mu.
func (c *Map) clone() map[string]Template {
	old := c.load()
	next := make(map[string]Template, len(old)+1)
	maps.Copy(next, old)

	return next
}

func (c *Map) Set(k string, v Template) {
	c.mu.Lock()
	defer c.mu.Unlock()

	next := c.clone()
	next[k] = v
	c.items.Store(&next)
}

// Get an item from the Map. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *Map) Get(k string) (Template, bool) {
	v, ok := c.load()[k]

	return v, ok
}

// Delete an item from the Map. Does nothing if the key is not in the Map.
func (c *Map) Delete(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.load()[k]; !ok {
		return // nothing to do, and no reason to pay for a copy
	}

	next := c.clone()
	delete(next, k)
	c.items.Store(&next)
}

// Clear resets the map.
func (c *Map) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items.Store(nil) // nil reads as empty
}
