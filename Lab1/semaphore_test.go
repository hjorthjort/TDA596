package main

import (
	"testing"
	"testing/synctest"
)

func TestAcquireSemaphore(t *testing.T) {
	sem, _ := CreateSemaphore(1)
	sem.Acquire()
}

func TestAcquireReleaseSemaphore(t *testing.T) {
	sem, _ := CreateSemaphore(1)
	sem.Acquire()
	sem.Release()
}

func TestSemaphoreBlocksUntilRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sem, _ := CreateSemaphore(2)
		sem.Acquire()
		sem.Acquire()

		acquired := make(chan struct{})
		go func() {
			sem.Acquire() // blocks
			close(acquired)
		}()
		synctest.Wait()

		select {
		// Acquire() blocks when all slots are taken
		case <-acquired:
			t.Fatal("third Acquire returned while both slots were occupied")
		default:
		}

		// Acquire() stops blocking when a new slot is opened
		sem.Release()
		synctest.Wait()
		select {
		case <-acquired:
		default:
			t.Fatal("Acquire remained blocked after Release freed a slot")
		}

		// release fully
		sem.Release()
		sem.Release()

		// reacquire fully
		sem.Acquire()
		sem.Acquire()

		// release partially
		sem.Release()
	})
}
