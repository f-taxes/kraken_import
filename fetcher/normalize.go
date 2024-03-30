package fetcher

var currenciesAliases = map[string]string{
	"ZEUR":     "EUR",
	"EUR.HOLD": "EUR",
	"XETH":     "ETH",
	"XXBT":     "XBT",
}

// Convert ambiguous currency strings to something that makes sense.
// There are, for example ZEUR and EUR.HOLD which both should simply be EUR.
func normalizeCurrency(v string) string {
	if c, ok := currenciesAliases[v]; ok {
		return c
	}

	return v
}
