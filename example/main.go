package main

import (
	"circuitbreaker"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// Simulated external service that fails randomly
func callExternalService() (string, error) {
	// Simulate 70% failure rate
	if rand.Float32() < 0.5 {
		return "", errors.New("service unavailable")
	}
	return "success", nil
}

func main() {
	// Create circuit breaker with custom settings
	cb := circuitbreaker.New(circuitbreaker.Settings{
		FailureThreshold: 3,               // Open after 3 failures
		SuccessThreshold: 2,               // Close after 2 successes in HalfOpen
		Timeout:          5 * time.Second, // Wait 5s before trying HalfOpen
		MaxRequests:      1,               // Allow 1 request in HalfOpen
		OnStateChange: func(from, to circuitbreaker.State) {
			fmt.Printf("🔄 Circuit state changed: %s → %s\n", from, to)
		},
	})

	fmt.Println("Starting Circuit Breaker Demo...")
	fmt.Println("================================")

	// Make 20 requests
	for i := 1; i <= 100; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			return callExternalService()
		})

		failures, successes := cb.Counts()

		if err != nil {
			fmt.Printf("Request %2d: ❌ Error: %s (failures: %d, successes: %d, state: %s)\n",
				i, err.Error(), failures, successes, cb.State())
		} else {
			fmt.Printf("Request %2d: ✅ Result: %s (failures: %d, successes: %d, state: %s)\n",
				i, result.(string), failures, successes, cb.State())
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Println("================================")
	fmt.Println("Demo finished!")
}
