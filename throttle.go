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
// Rate and Res can be used in conjection to give you a run frequency.
// For example rate = 10, res = time.Second will run something 10 times
// every second.
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
func (r *Runner) Do(ctx context.Context, total int, f func() error) error {
	var wg sync.WaitGroup
	wg.Add(total)

	errors := make(chan error)
	for i := 0; i < total; i++ {
		if r.rate > 0 {
			<-r.c
		}

		go func() {
			defer wg.Done()
			if err := f(); err != nil {
				errors <- err
			}
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
			return nil
		case <-ctx.Done():
			return nil
		case err := <-errors:
			return err
		}
	}
}

// DoFor executes a function for a given amount of time.  For example,
// if your throttler is configured to run 10 operations per second and
// you pass 3 seconds for d, this will execute the function 30 times.
func (r *Runner) DoFor(ctx context.Context, d time.Duration, f func() error) error {
	if d == 0 {
		return nil
	}

	end := time.After(d)
	errors := make(chan error)
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case <-end:
			wg.Wait()
			return nil
		case err := <-errors:
			return err
		default:
			if r.rate > 0 {
				<-r.c
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := f(); err != nil {
					errors <- err
				}
			}()
		}
	}
}

func qos(rate int64, res time.Duration) time.Duration {
	micros := res.Nanoseconds()
	return time.Duration(micros/rate) * time.Nanosecond
}
