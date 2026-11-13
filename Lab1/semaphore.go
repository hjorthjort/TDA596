package main

type Semaphore struct {
	sem chan struct{}
}

func CreateSemaphore(n int) *Semaphore {
	if n <= 0 {
		n = 1
	}
	sem := make(chan struct{}, n)
	return &Semaphore{sem: sem}
}

func (s *Semaphore) Acquire() {
	s.sem <- struct{}{}
}

// TryAcquire acquires a slot without waiting. It returns false when no slot is free.
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Semaphore) Release() {
	<-s.sem
}
