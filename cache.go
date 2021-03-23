package plates

import (
	"sync"
)

// Map is a thread-safe cache for templates
type Map struct {
	items map[string]Template
	sync.RWMutex
}

func NewMap() *Map {
	c := &Map{
		items: map[string]Template{},
	}

	return c
}

func (c *Map) Set(k string, v Template) {
	c.Lock()
	c.items[k] = v
	c.Unlock()
}

// Get an item from the Map. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *Map) Get(k string) (Template, bool) {
	c.RLock()
	v, ok := c.items[k]
	c.RUnlock()
	return v, ok
}

// Delete an item from the Map. Does nothing if the key is not in the Map.
func (c *Map) Delete(k string) {
	c.Lock()
	delete(c.items, k)
	c.Unlock()
}

// Clear resets the map.
func (c *Map) Clear() {
	c.Lock()
	c.items = map[string]Template{}
	c.Unlock()
}
