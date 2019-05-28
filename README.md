# throttle
Provides a simple interface for throttling function calls.

## Installation

``` bash
$ go get -u github.com/codingconcepts/throttle
```

## Usage

``` go
package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/codingconcepts/throttle"
)

func main() {
	// Create a new throttle of 10 ops/s.
	r := throttle.New(10, time.Second)

	var sum int64
	f := func() {
		atomic.AddInt64(&sum, 1)
	}

	// Run 20 times (takes 2s because we're running 10 ops/s).
	r.Do(context.Background(), 20, f)
	fmt.Printf("sum: %d\n", sum)
	// Outputs: 20

	// Run for 3 seconds.
	r.DoFor(context.Background(), time.Second*3, f)
	fmt.Printf("sum: %d\n", sum)
	// Outputs: 50
}
```
