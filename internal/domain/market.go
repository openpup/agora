package domain

import "fmt"

type Market string

const (
	MarketUSStock Market = "us_stock"
	MarketAStock  Market = "a_stock"
	MarketCrypto  Market = "crypto"
)

func (m Market) Valid() bool {
	switch m {
	case MarketUSStock, MarketAStock, MarketCrypto:
		return true
	default:
		return false
	}
}

func ParseMarket(value string) (Market, error) {
	m := Market(value)
	if !m.Valid() {
		return "", fmt.Errorf("invalid market: %s", value)
	}
	return m, nil
}
