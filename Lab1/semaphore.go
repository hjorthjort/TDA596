package main

import "errors"

type Semaphore struct {
	sem chan struct{}
}

func CreateSemaphore(n int) (Semaphore, error) {
	if n <= 0 {
		return Semaphore{}, errors.New("semaphore size must be greater than 0")
	}
	sem := make(chan struct{}, n)
	return Semaphore{sem: sem}, nil
}

func (s Semaphore) Acquire() {
	s.sem <- struct{}{}
}

func (s Semaphore) Release() {
	<-s.sem
}
