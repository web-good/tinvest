package semaphore

type Semaphore interface {
	Acquire()
	Release()
}

type semaphore struct {
	sem chan struct{}
}

func New(tickets int) Semaphore {
	return &semaphore{
		sem: make(chan struct{}, tickets),
	}
}

func (s *semaphore) Acquire() {
	s.sem <- struct{}{}
}

func (s *semaphore) Release() {
	<-s.sem
}
