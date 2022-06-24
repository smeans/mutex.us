package semaphore

import (
	"log"
	"math/rand"
	"sync"
	"time"
    "testing"
)

var semaphore *Semaphore
var workerWaitGroup sync.WaitGroup

func init() {
	semaphore = NewSemaphore(2)
}

func worker(n int) {
	defer workerWaitGroup.Done()

	lockWait := time.Duration((n-1.0)*100.0) * time.Millisecond
	log.Printf("worker %d waiting %s for lock", n, lockWait)
	if semaphore.Lock(lockWait, nil) != nil {
		log.Printf("worker %d failed to lock... abort", n)

		return
	}
	defer semaphore.Unlock()

	wait := time.Duration(rand.Float64()*3000.0+1000.0) * time.Millisecond
	log.Printf("worker %d acquired lock, waiting %s", n, wait)
	time.Sleep(wait)
	log.Printf("worker %d unlocking", n)
}

func startWorker(workerId int) {
	workerWaitGroup.Add(1)
	go worker(workerId)
}

func TestSemaphores(t *testing.T) {
	log.Println("test started")
	for i := 0; i < 5; i++ {
		startWorker(i)
	}

	workerWaitGroup.Wait()
	log.Println("test ended")
}
