package collection

import (
	"sort"
	"sync"
)

type collection[T any] struct {
	mu    sync.RWMutex
	items []T
}

func New[T any]() *collection[T] {
	return &collection[T]{items: make([]T, 0)}
}

func (c *collection[T]) Add(item T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = append(c.items, item)
}

func (c *collection[T]) GetTopByCriteria(fnCriteria func(i, j T) bool, size int) []T {
	copied := make([]T, len(c.items))
	copy(copied, c.items)
	sort.Slice(copied, func(i, j int) bool {
		return fnCriteria(copied[i], copied[j])
	})

	if len(copied) < size {
		return copied
	}

	return copied[:size]
}

func (c *collection[T]) GetAll() []T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Можно возвращать копию слайса, чтобы избежать гонки при чтении
	copied := make([]T, len(c.items))
	copy(copied, c.items)

	return copied
}

type Instrument struct {
	ID              string
	Name            string
	RSILength       int
	AverageDevident float64
}

type InstrumentCollection struct {
	instrument map[string]Instrument
}

func NewCollection() *InstrumentCollection {
	return &InstrumentCollection{
		instrument: make(map[string]Instrument),
	}
}

func (cc *InstrumentCollection) Add(instrument Instrument) *InstrumentCollection {
	cc.instrument[instrument.ID] = instrument

	return cc
}

func (cc *InstrumentCollection) Get(id string) (Instrument, bool) {
	company, exists := cc.instrument[id]
	return company, exists
}

func (cc *InstrumentCollection) Remove(id string) {
	delete(cc.instrument, id)
}

func (cc *InstrumentCollection) All() []Instrument {
	result := make([]Instrument, 0, len(cc.instrument))

	for _, company := range cc.instrument {
		result = append(result, company)
	}

	return result
}
