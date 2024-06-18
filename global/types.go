package global

import (
	"sort"

	"github.com/f-taxes/kraken_import/krakenapi"
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

type LedgerRecList []LedgerRec

func (e LedgerRecList) Sort() {
	sort.Slice(e, func(i, j int) bool {
		return e[i].Time < e[j].Time
	})
}

type LedgerRec struct {
	LedgerInfoDoc
	ID string
}

type TradeRec struct {
	krakenapi.TradeHistoryInfo
	LedgerRecs LedgerRecList
	ID         string
}

type LedgerInfoDoc struct {
	RefID   string  `json:"refid"`
	Time    float64 `json:"time"`
	Type    string  `json:"type"`
	Aclass  string  `json:"aclass"`
	Asset   string  `json:"asset"`
	Amount  string  `json:"amount"`
	Fee     string  `json:"fee"`
	Balance string  `json:"balance"`
}
