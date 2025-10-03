package domain

import (
	"sync"
	"tinvest/internal/domain/atr"
)

type Info struct {
	mu    sync.RWMutex
	items map[string]Item
}

type Item struct {
	InstrumentName string
	MACD           []*MACDItemTechAnalyse
	RSI            []*RSIItemTechAnalyse
	ATR            atr.ItemTechAnalyse
	RSIValue       int
	ProcentPrice   float64
}

func NewInfo() *Info {
	return &Info{
		mu:    sync.RWMutex{},
		items: make(map[string]Item),
	}
}

func (i *Info) WriteToMap(key string, value Item) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.items[key] = value
}

func (i *Info) ReadFromMap(key string) Item {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.items[key]
}

func (i *Info) Items() map[string]Item {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.items
}
