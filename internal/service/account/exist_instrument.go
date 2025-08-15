package account

func (a *account) ExistInstrument(instrumentID string) bool {
	_, ok := a.account.OrderLines[instrumentID]

	if ok {
		return true
	}

	return false
}
