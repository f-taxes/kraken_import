package fetcher

import (
	"github.com/shopspring/decimal"
)

var currenciesAliases = map[string]string{
	"ZEUR":     "EUR",
	"ZUSD":     "USD",
	"EUR.HOLD": "EUR",
	"XETH":     "ETH",
	"XXBT":     "BTC",
	"XBT":      "BTC",
}

// Convert ambiguous currency strings to something that makes sense.
// There are, for example ZEUR and EUR.HOLD which both should simply be EUR.
func normalizeCurrency(v string) string {
	if c, ok := currenciesAliases[v]; ok {
		return c
	}

	return v
}

func roundByCurrency(currency string, val float64) string {
	switch currency {
	case "EUR", "USD":
		return decimal.NewFromFloat(val).Round(4).String()
	default:
		return decimal.NewFromFloat(val).String()
	}
}
