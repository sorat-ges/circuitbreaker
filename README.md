# Circuit Breaker

A simple and lightweight Circuit Breaker implementation in Go.

## Installation

```bash
go get github.com/sorat-ges/circuitbreaker
```

## Usage

```go
package main

import (
    "circuitbreaker"
    "fmt"
    "net/http"
    "time"
)

func main() {
    cb := circuitbreaker.New(circuitbreaker.Settings{
        FailureThreshold: 5,               // Open after 5 failures
        SuccessThreshold: 2,               // Close after 2 successes in half-open
        Timeout:          30 * time.Second, // Wait before trying half-open
        MaxRequests:      1,               // Max requests in half-open state
        OnStateChange: func(from, to circuitbreaker.State) {
            fmt.Printf("Circuit: %s → %s\n", from, to)
        },
    })

    result, err := cb.Execute(func() (interface{}, error) {
        resp, err := http.Get("https://api.example.com/data")
        if err != nil {
            return nil, err
        }
        return resp, nil
    })

    if err != nil {
        fmt.Println("Error:", err)
        return
    }

    fmt.Println("Result:", result)
}
```

## States

| State | Description |
|-------|-------------|
| **Closed** | Normal operation, requests flow through |
| **Open** | Circuit is open, requests are blocked |
| **Half-Open** | Testing if the service has recovered |

## State Diagram

```
         ┌──────────────────────────────────────┐
         │                                      │
         ▼                                      │
    ┌─────────┐   fail >= threshold    ┌──────────┐
    │ CLOSED  │ ───────────────────────▶│  OPEN   │
    └─────────┘                        └──────────┘
         ▲                                   │
         │                                   │ timeout
         │                                   ▼
         │  success >= threshold     ┌────────────┐
         └───────────────────────────│ HALF-OPEN │
                                     └────────────┘
                                           │
                                           │ fail
                                           ▼
                                     back to OPEN
```

## Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `FailureThreshold` | 5 | Number of failures before opening the circuit |
| `SuccessThreshold` | 2 | Number of successes in half-open before closing |
| `Timeout` | 30s | Duration the circuit stays open before half-open |
| `MaxRequests` | 1 | Max requests allowed in half-open state |
| `OnStateChange` | nil | Callback when state changes |

## API

```go
type CircuitBreaker interface {
    Execute(fn func() (interface{}, error)) (interface{}, error)
    State() State
    Reset()
    Counts() (failures int, successes int)
}
```

## License

MIT
