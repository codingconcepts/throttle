package throttle

import (
	"errors"
	"log"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	cases := []struct {
		name   string
		rps    int64
		res    time.Duration
		total  int
		exp    int64
		expErr bool
	}{
		{name: "no throttle without requests", rps: 0, res: time.Millisecond, total: 0, exp: 0},
		{name: "1/ms throttle without requests", rps: 1, res: time.Millisecond, total: 0, exp: 0},
		{name: "no throttle with 1 request", rps: 0, res: time.Millisecond, total: 1, exp: 1},
		{name: "1/ms throttle with 1 request", rps: 1, res: time.Millisecond, total: 1, exp: 1},
		{name: "10/ms throttle with 1 request", rps: 10, res: time.Millisecond, total: 1, exp: 1},
		{name: "10/ms throttle with 10 requests", rps: 10, res: time.Millisecond, total: 10, exp: 10},
		{name: "error", rps: 1, res: time.Millisecond, total: 1, exp: 0, expErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := New(c.rps, c.res)

			var sum int64
			err := r.Do(c.total, func() error {
				if c.expErr {
					return errors.New("bang")
				}
				atomic.AddInt64(&sum, 1)
				return nil
			})

			if err != nil && !c.expErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && c.expErr {
				t.Fatal("expected error but didn't get one")
			}

			equals(t, c.exp, sum)
		})
	}
}

func TestDoFor(t *testing.T) {
	cases := []struct {
		name   string
		rps    int64
		res    time.Duration
		d      time.Duration
		exp    int64
		expErr bool
	}{
		{name: "no throttle without requests", rps: 0, res: time.Millisecond, d: 0, exp: 0},
		{name: "1 throttle for 1ms", rps: 1, res: time.Millisecond, d: time.Millisecond, exp: 1},
		{name: "1 throttle for 2ms", rps: 10, res: time.Millisecond, d: time.Millisecond * 2, exp: 20},
		{name: "10 throttle with 1ms", rps: 10, res: time.Millisecond, d: time.Millisecond, exp: 10},
		{name: "error", rps: 1, res: time.Millisecond, d: time.Millisecond, expErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := New(c.rps, c.res)

			var sum int64
			err := r.DoFor(c.d, func() error {
				if c.expErr {
					return errors.New("bang")
				}
				atomic.AddInt64(&sum, 1)
				return nil
			})

			if err != nil && !c.expErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && c.expErr {
				t.Fatal("expected error but didn't get one")
			}

			equals(t, c.exp, sum)
		})
	}
}

func TestQOS(t *testing.T) {
	cases := []struct {
		name string
		rate int64
		res  time.Duration
		exp  time.Duration
	}{
		{name: "1/h", rate: 1, res: time.Hour, exp: time.Hour},
		{name: "1/m", rate: 1, res: time.Minute, exp: time.Minute},
		{name: "1/s", rate: 1, res: time.Second, exp: time.Second},
		{name: "1/ms", rate: 1, res: time.Millisecond, exp: time.Millisecond},
		{name: "1/µs", rate: 1, res: time.Microsecond, exp: time.Microsecond},
		{name: "60/h", rate: 60, res: time.Hour, exp: time.Minute},
		{name: "60/m", rate: 60, res: time.Minute, exp: time.Second},
		{name: "1000/s", rate: 1000, res: time.Second, exp: time.Millisecond},
		{name: "1000/ms", rate: 1000, res: time.Millisecond, exp: time.Microsecond},
		{name: "1000/µs", rate: 1000, res: time.Microsecond, exp: time.Nanosecond},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := qos(c.rate, c.res)
			equals(t, c.exp, act)
		})
	}
}

func TestTotal(t *testing.T) {
	cases := []struct {
		name string
		rate int64
		res  time.Duration
		d    time.Duration
		exp  int64
	}{
		{name: "1/s for 1s", rate: 1, res: time.Second, d: time.Second, exp: 1},
		{name: "2/s for 1s", rate: 2, res: time.Second, d: time.Second, exp: 2},
		{name: "1/s for 2s", rate: 1, res: time.Second, d: time.Second * 2, exp: 2},
		{name: "1/ms for 1s", rate: 1, res: time.Millisecond, d: time.Second, exp: 1000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := total(c.rate, c.res, c.d)
			if act != c.exp {
				t.Fatalf("exp: %d\ngot: %d\n", c.exp, act)
			}
		})
	}
}

func Example() {
	r := New(10, time.Second)

	var sum int64
	r.Do(10, func() error {
		atomic.AddInt64(&sum, 1)
		return nil
	})
	log.Println("sum", sum)
}

func equals(tb testing.TB, exp, act interface{}) {
	tb.Helper()
	if !reflect.DeepEqual(exp, act) {
		tb.Fatalf("\n\texp: %#[1]v (%[1]T)\n\tgot: %#[2]v (%[2]T)\n", exp, act)
	}
}