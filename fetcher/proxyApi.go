package fetcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	g "github.com/f-taxes/kraken_import/global"
	"github.com/f-taxes/kraken_import/krakenapi"
	"go.uber.org/ratelimit"
)

type ProxyApi struct {
	cacheFolder string
	realApi     *krakenapi.KrakenAPI
	cacheKey    string
	limiter     ratelimit.Limiter
}

func NewProxyApi(label, key, secret string) *ProxyApi {
	return &ProxyApi{
		realApi:     krakenapi.New(key, secret),
		cacheKey:    label,
		cacheFolder: "./cache",
		limiter:     ratelimit.New(8, ratelimit.Per(time.Minute)),
	}
}

func (a *ProxyApi) ensureCache() {
	os.MkdirAll(a.cacheFolder, 0755)
}

func (a *ProxyApi) readCache(key string) []byte {
	a.ensureCache()
	p := filepath.Join(a.cacheFolder, fmt.Sprintf("%s_%s.json", a.cacheKey, key))
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}

	return data
}

func (a *ProxyApi) writeCache(key string, data []byte) error {
	a.ensureCache()
	p := filepath.Join(a.cacheFolder, fmt.Sprintf("%s_%s.json", a.cacheKey, key))
	err := os.WriteFile(p, data, 0755)
	if err != nil {
		return err
	}

	return nil
}

func (a *ProxyApi) TradesHistory(start int64, end int64, args map[string]string) (*krakenapi.TradesHistoryResponse, error) {
	cacheName := fmt.Sprintf("trades_%d_%d_%s", start, end, args["ofs"])

	if cached := a.readCache(cacheName); cached != nil {
		resp := krakenapi.TradesHistoryResponse{}
		err := json.Unmarshal(cached, &resp)
		if err != nil {
			return nil, err
		}

		return &resp, nil
	}

	a.limiter.Take()
	resp, err := a.realApi.TradesHistory(start, end, args)

	if err == nil {
		data, err := json.Marshal(*resp)
		if err != nil {
			return nil, err
		}
		a.writeCache(cacheName, data)
	}

	return resp, err
}

func (a *ProxyApi) Ledgers(args map[string]string) (map[string]g.LedgerInfoDoc, error) {
	cacheName := fmt.Sprintf("ledgers_%s_%s", args["start"], args["ofs"])

	if cached := a.readCache(cacheName); cached != nil {
		resp := map[string]g.LedgerInfoDoc{}
		err := json.Unmarshal(cached, &resp)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	a.limiter.Take()
	resp, err := a.realApi.Ledgers(args)
	recs := map[string]g.LedgerInfoDoc{}

	if err == nil {

		for key, entry := range resp.Ledger {
			recs[key] = g.LedgerInfoDoc{
				RefID:   entry.RefID,
				Time:    entry.Time,
				Type:    entry.Type,
				Aclass:  entry.Aclass,
				Asset:   entry.Asset,
				Amount:  entry.Amount.Text('f', 8),
				Fee:     entry.Fee.Text('f', 8),
				Balance: entry.Balance.Text('f', 8),
			}
		}

		data, err := json.Marshal(recs)
		if err != nil {
			return nil, err
		}
		a.writeCache(cacheName, data)
	}

	return recs, err
}
