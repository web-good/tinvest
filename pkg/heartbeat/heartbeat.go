package heartbeat

import (
	"time"
)

type heartbeat struct {
	doneCh chan struct{}
}

func (h *heartbeat) Beating(tickerTime time.Duration) chan struct{} {
	heartBeatCh := make(chan struct{})
	go func() {
		pulse := time.NewTicker(tickerTime)

		defer func() {
			close(heartBeatCh)
			pulse.Stop()
		}()

		for {
			select {
			case _, ok := <-h.doneCh:
				if !ok {
					return
				}
			case <-pulse.C:
				heartBeatCh <- struct{}{}
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return heartBeatCh
}

func (h *heartbeat) Stop() {
	close(h.doneCh)
}

func New() *heartbeat {
	return &heartbeat{
		doneCh: make(chan struct{}),
	}
}
