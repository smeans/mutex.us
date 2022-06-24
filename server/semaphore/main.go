package semaphore

import (
	"time"
	"errors"
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

func (s *Semaphore) Lock(timeout time.Duration, done <-chan struct{}) error {
	timeoutChannel := make(chan bool, 1)

	if timeout >= 0 {
		go func() {
			time.Sleep(timeout)
			timeoutChannel <- true
		}()
	}

	select {
	case _, ok := <-s.resource:
		if ok {
			return nil
		} else {
			return errors.New("lock failed: unable to acquire lock")
		}
	case <-done:
		return errors.New("lock failed: client disconnected")
	case _ = <-timeoutChannel:
		return errors.New("lock failed: wait timeout expired")
	}

	return errors.New("lock failed: unknown error")
}

func (s *Semaphore) Unlock() bool {
	select {
	case s.resource <- true:
		return true
	default:
		return false
	}
}
