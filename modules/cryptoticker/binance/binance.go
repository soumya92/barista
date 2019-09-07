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

// Package binance provides crypto coin ticker using the Binance API
// available at https://api.binance.com.
package binance // import "barista.run/modules/cryptoticker/binance"

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"barista.run/modules/cryptoticker"
)

// binananceCryptoTicker represents an binance ticker json response.
type binanceCryptoTicker struct {
	Symbol				string		`json:"symbol"`
	PriceChange			float32		`json:"priceChange,string"`
	PriceChangePercent	float32		`json:"priceChangePercent,string"`
	WeightedAvgPrice	float32		`json:"weightedAvgPrice,string"`
	PrevClosePrice		float32		`json:"prevClosePrice,string"`
	LastPrice			float32		`json:"lastPrice,string"`
	LastQty				float32		`json:"lastQty,string"`
	BidPrice			float32		`json:"bidPrice,string"`
	AskPrice			float32		`json:"askPrice,string"`
	OpenPrice			float32		`json:"openPrice,string"`
	HighPrice			float32		`json:"highPrice,string"`
	LowPrice			float32		`json:"lowPrice,string"`
	Volume				float32		`json:"volume,string"`
	QuoteVolume			float32		`json:"quoteVolume,string"`
	OpenTime			uint64		`json:"openTime"`
	CloseTime			uint64		`json:"closeTime"`
	FirstID				uint64		`json:"firstId"`
	LastID				uint64		`json:"lastId"`
	Count				uint64		`json:"count"`
}

// Provider wraps a coin symbol so that it can be used as a cryptoticker.Provider.
type Provider struct{
	Symbol string
}

// GetTicker gets ticker information from Binance.
func (binance Provider) GetTicker() (cryptoticker.CryptoTicker, error) {
	srvURL := url.URL{
		Scheme:		"https",
		Host:		"api.binance.com",
		Path:		"/api/v3/ticker/24hr",
		RawQuery:	"symbol=" + binance.Symbol,
	}

	response, err := http.Get(srvURL.String())
	if err != nil {
		return cryptoticker.CryptoTicker{}, err
	}
	defer response.Body.Close()
	o := binanceCryptoTicker{}
	err = json.NewDecoder(response.Body).Decode(&o)
	if err != nil {
		return cryptoticker.CryptoTicker{}, err
	}
	if o.Symbol == "" {
		return cryptoticker.CryptoTicker{}, fmt.Errorf("Bad response from Binance")
	}

	return cryptoticker.CryptoTicker{
		Symbol:				o.Symbol,
		PriceChange:		o.PriceChange,
		PriceChangePercent:	o.PriceChangePercent,
		PrevClosePrice:		o.PrevClosePrice,
		LastPrice:			o.LastPrice,
		LastQty:			o.LastQty,
		OpenPrice:			o.OpenPrice,
		HighPrice:			o.HighPrice,
		LowPrice:			o.LowPrice,
		Volume:				o.Volume,
		OpenTime:			o.OpenTime,
		CloseTime:			o.CloseTime,
		Attribution:		"Binance",
	}, nil
}
