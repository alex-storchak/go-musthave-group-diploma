package ticker

import "time"

// Factory создает Ticker с заданным интервалом.
type Factory interface {
	New(d time.Duration) Ticker
}

type RealTickerFactory struct{}

func (RealTickerFactory) New(d time.Duration) Ticker {
	return &RealTicker{t: time.NewTicker(d)}
}

type FakeTickerFactory struct {
	T *FakeTicker
}

func (f *FakeTickerFactory) New(_ time.Duration) Ticker {
	// d игнорируем — тест сам управляет тиками
	return f.T
}
