// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cryptoticker provides an i3bar module that displays crypto coin
// ticker info.
package cryptoticker // import "barista.run/modules/cryptoticker"

import (
	"time"

	"barista.run/bar"
	"barista.run/base/notifier"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"
	"barista.run/timing"
)

// CryptoTicker represents the crypto coin ticker.
type CryptoTicker struct {
	Symbol				string
	PriceChange			float32
	PriceChangePercent	float32
	PrevClosePrice		float32
	LastPrice			float32
	LastQty				float32
	OpenPrice			float32
	HighPrice			float32
	LowPrice			float32
	Volume				float32
	OpenTime			uint64
	CloseTime			uint64
	Attribution			string
}

// Provider is an interface for cryptoticker providers.
type Provider interface {
	GetTicker() (CryptoTicker, error)
}

// Module represents a bar.Module that displays crypto coin information.
type Module struct {
	provider   Provider
	scheduler  *timing.Scheduler
	refreshFn  func()
	refreshCh  <-chan struct{}
	outputFunc value.Value // of func(Ticker) bar.Output
}

// New constructs an instance of the cryptoticker module with the provided configuration.
func New(provider Provider) *Module {
	m := &Module{
		provider:  provider,
		scheduler: timing.NewScheduler(),
	}
	m.refreshFn, m.refreshCh = notifier.New()
	l.Register(m, "outputFunc", "clickHandler", "scheduler")

	// Default output is symbol, last price and percent change.
	m.Output(func(t CryptoTicker) bar.Output {
		return outputs.Textf("%s %.2f %+.2f%%",
			t.Symbol, t.LastPrice, t.PriceChangePercent)
	})
	m.RefreshInterval(5 * time.Second)
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(CryptoTicker) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RefreshInterval configures the polling frequency.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// Refresh fetches updated ticker information.
func (m *Module) Refresh() {
	m.refreshFn()
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	ticker, err := m.provider.GetTicker()
	outputFunc := m.outputFunc.Get().(func(CryptoTicker) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()

	defer done()

	for {
		if !s.Error(err) {
			s.Output(outputFunc(ticker))
		}
		select {
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(CryptoTicker) bar.Output)
		case <-m.scheduler.C:
			ticker, err = m.provider.GetTicker()
		case <-m.refreshCh:
			if err != nil {
				s(nil)
			}
			ticker, err = m.provider.GetTicker()
		}
	}
}
