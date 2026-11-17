package service_provider

import (
	"strings"
	"sync"
	"tinvest/internal/service/screener/density"
)

// densityAlerts retains ticker admission for this application run. It never
// carries orders or execution state, so delivery cannot block position exits.
type densityAlerts struct {
	mu  sync.RWMutex
	ids map[string]bool
}

func (a *densityAlerts) Record(s density.Signal) {
	if s.InstrumentID == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ids == nil {
		a.ids = make(map[string]bool)
	}
	a.ids[s.InstrumentID] = true
}
func (s *ServiceProvider) densityEntryFilter(tickers []string) func(density.Instrument, density.Signal) bool {
	fixed := make(map[string]bool, len(tickers))
	for _, t := range tickers {
		fixed[strings.ToUpper(strings.TrimSpace(t))] = true
	}
	return func(i density.Instrument, _ density.Signal) bool {
		if fixed[strings.ToUpper(i.Ticker)] {
			return true
		}
		a := &s.service.densityAlerts
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.ids[i.ID]
	}
}
