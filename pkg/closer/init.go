package closer

import (
	"os"
	"os/signal"
)

func initCloser(sig ...os.Signal) *Closer {
	c := &Closer{done: make(chan struct{})}

	if len(sig) <= 0 {
		return c
	}

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, sig...)
		<-ch
		signal.Stop(ch)
		c.CloseAll()
	}()

	return c
}
