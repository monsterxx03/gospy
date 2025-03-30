package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

func main() {
	// Goroutines in different states
	var wg sync.WaitGroup

	// Create goroutines in various states
	createRunningGoroutines(&wg, 5)
	createBlockedOnChannelGoroutines(&wg, 3)
	createBlockedOnMutexGoroutines(&wg, 2)
	createSleepingGoroutines(&wg, 4)
	createSystemGoroutine()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"arch": "%s", "pid": %d}`, runtime.GOARCH, os.Getpid())
	})

	fmt.Println("Server listening on :8080")
	fmt.Println("Created goroutines in different states:")
	fmt.Println("- 5 running (busy loop)")
	fmt.Println("- 3 blocked on channel")
	fmt.Println("- 2 blocked on mutex") 
	fmt.Println("- 4 sleeping")
	fmt.Println("- 1 system (GC)")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
		os.Exit(1)
	}

	wg.Wait() // Keep goroutines alive
}
// createRunningGoroutines creates goroutines in running state (busy loop)
func createRunningGoroutines(wg *sync.WaitGroup, count int) {
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				time.Sleep(time.Second)
			}
		}()
	}
}

// createBlockedOnChannelGoroutines creates goroutines blocked on a channel
func createBlockedOnChannelGoroutines(wg *sync.WaitGroup, count int) {
	ch := make(chan struct{})
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ch
		}()
	}
}

// createBlockedOnMutexGoroutines creates goroutines blocked on a mutex
func createBlockedOnMutexGoroutines(wg *sync.WaitGroup, count int) {
	var mu sync.Mutex
	mu.Lock()
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
		}()
	}
}

// createSleepingGoroutines creates goroutines in sleeping state
func createSleepingGoroutines(wg *sync.WaitGroup, count int) {
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Hour)
		}()
	}
}

// createSystemGoroutine creates system goroutines (like GC)
func createSystemGoroutine() {
	go func() {
		for {
			runtime.GC()
			time.Sleep(5 * time.Second)
		}
	}()
}
