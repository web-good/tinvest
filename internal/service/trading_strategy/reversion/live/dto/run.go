package dto

// Mode selects which pass a scheduled run executes.
type Mode int

const (
	ModeBuy Mode = iota
	ModeManage
)

// Run is one scheduled invocation of the reversion runner.
type Run struct {
	Scheduler string
	Mode      Mode
}
