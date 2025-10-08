package collection

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
