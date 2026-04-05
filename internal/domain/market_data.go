package domain

import "time"

type Candle struct {
	Time     time.Time      `json:"time"`
	Ticker   string         `json:"ticker"`
	Market   Market         `json:"market"`
	Open     float64        `json:"open"`
	High     float64        `json:"high"`
	Low      float64        `json:"low"`
	Close    float64        `json:"close"`
	Volume   float64        `json:"volume"`
	Metadata map[string]any `json:"metadata"`
}

type MarketDataQuery struct {
	Ticker   string
	Market   Market
	Interval string
	From     *time.Time
	To       *time.Time
}
