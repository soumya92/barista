// Copyright 2018 Google Inc.
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

package cryptoticker

import (
	"errors"
	"sync"
	"testing"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
)

type testProvider struct {
	sync.RWMutex
	CryptoTicker
	error
}

func (t *testProvider) GetTicker() (CryptoTicker, error) {
	t.RLock()
	defer t.RUnlock()
	return t.CryptoTicker, t.error
}

func TestTicker(t *testing.T) {
	testBar.New(t)
	p := &testProvider{CryptoTicker: CryptoTicker{
		Symbol:					"BTCUSDT",
		LastPrice:				10000.00,
		PriceChangePercent:		1.01,
		Attribution:			"TEST",
	}}
	w := New(p)
	testBar.Run(w)

	testBar.NextOutput().AssertText(
		[]string{"BTCUSDT 10000.00 +1.01%"}, "on start")

	testBar.Tick()
	testBar.NextOutput().Expect("on tick")

	w.Output(func(w CryptoTicker) bar.Output {
		return outputs.Textf("%s: %.2f", w.Symbol, w.LastPrice)
	})
	testBar.NextOutput().AssertText([]string{
		"BTCUSDT: 10000.00"}, "on template change")

	p.Lock()
	p.error = errors.New("foo")
	p.Unlock()

	testBar.Tick()
	testBar.NextOutput().AssertError("on tick with error")

	testBar.Tick()
	out := testBar.NextOutput("on tick with error")

	p.Lock()
	p.error = nil
	out.At(0).LeftClick()
	testBar.NextOutput().AssertEmpty("clears error on refresh")

	p.Unlock()
	testBar.NextOutput().AssertText([]string{"BTCUSDT: 10000.00"})
}
