package global

import (
	krakenapi "github.com/beldur/kraken-go-api-client"
)

type Account struct {
	ID        string `mapstructure:"id" json:"id"`
	Label     string `mapstructure:"label" json:"label"`
	Notes     string `mapstructure:"notes" json:"notes"`
	ApiKey    string `mapstructure:"key" json:"key"`
	ApiSecret string `mapstructure:"secret" json:"secret"`

	// Timestamp of the last time the plugin fetched trades from the source.
	LastFetched string `mapstructure:"lastFetched" json:"lastFetched"`
}

type LedgerRec struct {
	krakenapi.LedgerInfo
	ID string
}

type TradeRec struct {
	krakenapi.TradeHistoryInfo
	ID string
}
