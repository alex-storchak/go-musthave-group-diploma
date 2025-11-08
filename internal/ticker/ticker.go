package ticker

import (
	"sync"
	"time"
)

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// RealTicker — обертка над time.Ticker.
type RealTicker struct {
	t *time.Ticker
}

func (t *RealTicker) C() <-chan time.Time {
	return t.t.C
}
func (t *RealTicker) Stop() {
	t.t.Stop()
}

type FakeTicker struct {
	Ch   chan time.Time
	once sync.Once
}

func NewFakeTicker() *FakeTicker {
	return &FakeTicker{Ch: make(chan time.Time, 1024)}
}

func (f *FakeTicker) C() <-chan time.Time {
	return f.Ch
}

func (f *FakeTicker) Stop() {
	f.once.Do(
		func() {
			close(f.Ch)
		},
	)
}
