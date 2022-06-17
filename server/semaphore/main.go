package semaphore

import (
	"time"
)

type Semaphore struct {
	resource chan bool
}

func NewSemaphore(limit int) *Semaphore {
	if limit < 1 {
		return nil
	}

	s := new(Semaphore)
	s.resource = make(chan bool, limit)

	for i := 0; i < limit; i++ {
		s.resource <- true
	}

	return s
}

func (s *Semaphore) Lock(timeout time.Duration) bool {
	timeoutChannel := make(chan bool, 1)

	if timeout >= 0 {
		go func() {
			time.Sleep(timeout)
			timeoutChannel <- true
		}()
	}

	select {
	case _, ok := <-s.resource:
		return ok
	case _ = <-timeoutChannel:
		return false
	}

	return false
}

func (s *Semaphore) Unlock() bool {
	select {
	case s.resource <- true:
		return true
	default:
		return false
	}
}
