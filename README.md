# GoRace

The purpose of this package is to prevent race conditions that cause goroutine panics on long running api calls by encapsulating goroutine logic in a shielded and cancelable implementation. This package is designed specifically for use with contexts. This package removes the need to code complex goroutine and channel logic that would otherwise be required to prevent race conditions.

## Sample code:
```
package main

import (
        "context"
        "fmt"
        "time"

        "github.com/magic53/gorace"
)

func main() {
	logic()
        fmt.Println("end main")
}

// Wrap an api cancelable
func logic() {
	cancelable := gorace.GoRace(func(ctx context.Context, cancelable gorace.GoCancelable) {
		// Without the cancelable, this async function call could potential execute after
                // logic() returns causing a panic. Due to context deadline returning before the
		// API call is complete, In this case, work(). Instead the panic is avoided because
		// cancelable will not attempt to send on the closed channel
                if result, err := work(ctx); err != nil {
        		cancelable.Send(err)
        	} else {
                        cancelable.Send(result)
        	}
        })
        cancelable.StartBackground(context.Background())

        select {
        case result := <-cancelable.Receive():
                switch t := result.(type) {
                case error:
                        fmt.Printf("Receiving: %s\n", t.Error())
                case bool:
                        fmt.Printf("Receiving: %v\n", t)
                }
        case <-time.After(2*time.Second):
                fmt.Println("canceling")
                cancelable.Cancel()
        }
}

// Do some work
func work(ctx context.Context) (bool, error) {
        <-time.After(time.Second)
        return true, nil
}
```