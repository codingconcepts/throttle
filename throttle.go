package throttle

import (
	"context"
	"sync"
	"time"
)

// Runner holds the methods of the interface.
type Runner struct {
	rate int64
	res  time.Duration
	c    <-chan time.Time
}

// New returns a pointer to an instance of runner, which is used to
// perform all operations at the given rate in requests/s.
//
// Rate is the number of requests you wish to run
func New(rate int64, res time.Duration) *Runner {
	r := Runner{
		rate: rate,
		res:  res,
	}

	if rate > 0 {
		r.c = time.NewTicker(qos(rate, res)).C
	}

	return &r
}

// Do executes a function a given number of times.  For example, if
// your throttler is configured to run 10 operations per second and
// you pass 50 for total, this will execute the function 50 times
// and take 5 seconds.
func (r *Runner) Do(ctx context.Context, total int, f func()) {
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		if r.rate > 0 {
			<-r.c
		}

		go func() {
			defer wg.Done()
			f()
		}()
	}

	// Allow context cancellations to be provided.
	finished := make(chan struct{})
	go func() {
		wg.Wait()
		finished <- struct{}{}
	}()

	for {
		select {
		case <-finished:
			return
		case <-ctx.Done():
			return
		}
	}
}

// DoFor executes a function for a given amount of time.  For example,
// if your throttler is configured to run 10 operations per second and
// you pass 3 seconds for d, this will execute the function 30 times.
func (r *Runner) DoFor(ctx context.Context, d time.Duration, f func()) {
	if d == 0 {
		return
	}

	end := time.After(d)
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-end:
			wg.Wait()
			return
		default:
			if r.rate > 0 {
				<-r.c
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				f()
			}()
		}
	}
}

func qos(rate int64, res time.Duration) time.Duration {
	micros := res.Nanoseconds()
	return time.Duration(micros/rate) * time.Nanosecond
}
