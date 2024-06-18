package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	g "github.com/f-taxes/kraken_import/global"
	"github.com/f-taxes/kraken_import/grpc_client"
	"github.com/f-taxes/kraken_import/proto"
	"github.com/kataras/golog"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/ratelimit"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Fetcher struct {
	label      string
	restClient *ProxyApi
	assets     map[string]AssetInfo
	pairs      map[string]PairInfo
}

func New(label, key, secret string) (*Fetcher, error) {
	client := NewProxyApi(label, key, secret)

	f := &Fetcher{
		label:      label,
		restClient: client,
	}

	f.LoadAssets()

	return f, f.LoadPairs()
}

var limiter = ratelimit.New(8, ratelimit.Per(time.Minute))

func (f *Fetcher) LoadAssets() error {
	limiter.Take()

	resp, err := http.Get("https://api.kraken.com/0/public/Assets")
	if err != nil {
		return err
	}
	data, _ := io.ReadAll(resp.Body)

	var assetResp AssetsResponse
	err = json.Unmarshal(data, &assetResp)
	if err != nil {
		return err
	}

	f.assets = assetResp.Result

	return nil
}

func (f *Fetcher) LoadPairs() error {
	limiter.Take()

	resp, err := http.Get("https://api.kraken.com/0/public/AssetPairs")
	if err != nil {
		return err
	}
	data, _ := io.ReadAll(resp.Body)

	var pairResponse PairResponse
	err = json.Unmarshal(data, &pairResponse)
	if err != nil {
		return err
	}

	f.pairs = pairResponse.Result

	return nil
}

func (f *Fetcher) findLedgerRecs(legerIDList []string, ledgerRecs g.LedgerRecList) g.LedgerRecList {
	matches := g.LedgerRecList{}
	for _, id := range legerIDList {
		for i := range ledgerRecs {
			if ledgerRecs[i].ID == id {
				matches = append(matches, ledgerRecs[i])
			}
		}
	}

	matches.Sort()

	return matches
}

func (f *Fetcher) ledgerRecByCurrency(currency string, ledgerRecs g.LedgerRecList) *g.LedgerRec {
	for i := range ledgerRecs {
		if ledgerRecs[i].Asset == currency {
			return &ledgerRecs[i]
		}
	}

	return nil
}

func (f *Fetcher) Trades(lastFetched time.Time, ledgerRecs []g.LedgerRec) error {
	lastFetched = time.Time{}

	if len(ledgerRecs) == 0 {
		d, err := os.ReadFile("ledger.json")
		if err != nil {
			return err
		}
		err = json.Unmarshal(d, &ledgerRecs)
		if err != nil {
			return err
		}
	} else {
		d, _ := json.MarshalIndent(ledgerRecs, "", "  ")
		os.WriteFile("ledger.json", d, 0775)
	}

	jobId := primitive.NewObjectID().Hex()
	grpc_client.GrpcClient.ShowJobProgress(context.Background(), &proto.JobProgress{
		ID:       jobId,
		Label:    fmt.Sprintf("Fetching newest trades for account \"%s\"", f.label),
		Progress: "-1",
	})

	defer grpc_client.GrpcClient.ShowJobProgress(context.Background(), &proto.JobProgress{
		ID:       jobId,
		Progress: "100",
	})

	start := lastFetched.Unix()
	seen := map[string]struct{}{}
	count := 0
	page := 0

outer:
	for {
		params := map[string]string{
			"ofs":     fmt.Sprintf("%d", page),
			"ledgers": "true",
		}

		resp, err := f.restClient.TradesHistory(start, 0, params)

		if err != nil {
			return err
		}

		recs := []g.TradeRec{}

		for lid, entry := range resp.Trades {
			if _, ok := seen[lid]; !ok {
				seen[lid] = struct{}{}
				recs = append(recs, g.TradeRec{TradeHistoryInfo: entry, ID: lid, LedgerRecs: f.findLedgerRecs(entry.Ledgers, ledgerRecs)})
			}
		}

		if len(recs) == 0 {
			break outer
		}

		page += len(recs)

		sort.Slice(recs, func(i, j int) bool {
			return recs[i].Time >= recs[j].Time
		})

		for i := range recs {
			r := recs[i]
			ts := time.Unix(int64(r.Time), 0).UTC()

			if r.ID == "TAPYVU-AZ24B-JGVUZX" {
				fmt.Println(r.ID)
			}

			count++

			pair, ok := f.pairs[r.AssetPair]
			if !ok {
				grpc_client.GrpcClient.AppLog(context.Background(), &proto.AppLogMsg{Level: proto.LogLevel_ERR, Message: fmt.Sprintf("[%s] Pair %s wasn't found in krakens list of pairs. This shouldn't be happening.", g.Plugin.Label, r.AssetPair)})
				return nil
			}

			baseAsset, ok := f.assets[pair.Base]
			if !ok {
				grpc_client.GrpcClient.AppLog(context.Background(), &proto.AppLogMsg{Level: proto.LogLevel_ERR, Message: fmt.Sprintf("[%s] Asset %s wasn't found in krakens list of assets. This shouldn't be happening.", g.Plugin.Label, pair.Base)})
				return nil
			}

			quoteAsset, ok := f.assets[pair.Quote]
			if !ok {
				grpc_client.GrpcClient.AppLog(context.Background(), &proto.AppLogMsg{Level: proto.LogLevel_ERR, Message: fmt.Sprintf("[%s] Asset %s wasn't found in krakens list of assets. This shouldn't be happening.", g.Plugin.Label, pair.Base)})
				return nil
			}

			side := proto.TxAction_BUY

			if r.Type == "sell" {
				side = proto.TxAction_SELL
			}

			orderType := proto.OrderType_TAKER

			if r.OrderType == "limit" {
				orderType = proto.OrderType_MAKER
			}

			baseAssetNormalized := normalizeCurrency(baseAsset.Altname)
			quoteAssetNormalized := normalizeCurrency(quoteAsset.Altname)
			// cost := roundByCurrency(quoteAssetNormalized, r.Cost)
			// fee := roundByCurrency(quoteAssetNormalized, r.Fee)

			// ledgerRecBase := f.ledgerRecByCurrency(pair.Base, r.LedgerRecs)
			// ledgerRecQuote := f.ledgerRecByCurrency(pair.Quote, r.LedgerRecs)

			amount := decimal.Zero
			value := fmt.Sprintf("%f", r.Cost)
			// fee := "0"
			// quoteFee := fmt.Sprintf("%f", r.Fee)
			// isMargin := false

			// if ledgerRecBase != nil {
			// 	fee = ledgerRecBase.Fee.Abs(&ledgerRecBase.Fee).String()
			// 	isMargin = isMargin || ledgerRecBase.Type == "margin"
			// }

			// if ledgerRecQuote != nil {
			// 	quoteFee = ledgerRecQuote.Fee.Abs(&ledgerRecQuote.Fee).String()
			// 	isMargin = isMargin || ledgerRecQuote.Type == "margin"
			// }

			fee := decimal.Zero
			quoteFee := decimal.Zero
			feeDecimals := 0
			quoteFeeDecimals := 0
			feeCurrency := baseAssetNormalized
			isMargin := false

			for _, l := range r.LedgerRecs {
				if l.Type == "margin" {
					isMargin = true
				}

				switch l.Asset {
				case pair.Base:
					amount = amount.Add(g.StrToDecimal(l.Amount).Abs())
					fee = fee.Add(g.StrToDecimal(l.Fee).Abs())
					feeDecimals = f.assets[l.Asset].Decimals
				case pair.Quote:
					quoteFee = quoteFee.Add(g.StrToDecimal(l.Fee).Abs())
					quoteFeeDecimals = f.assets[l.Asset].Decimals
				default:
					if isMargin {
						feeCurrency = normalizeCurrency(l.Asset)
						fee = fee.Add(g.StrToDecimal(l.Fee).Abs())
					}
				}
			}

			if amount.IsZero() {
				amount = decimal.NewFromFloat(r.Volume)
			}

			props := &proto.TradeProps{
				IsMarginTrade: isMargin,
				IsPhysical:    true,
				IsDerivative:  false,
			}

			trade := &proto.Trade{
				TxID:             r.ID,
				Ts:               timestamppb.New(ts),
				Account:          f.label,
				Ticker:           pair.Wsname,
				Quote:            quoteAssetNormalized,
				Asset:            baseAssetNormalized,
				Price:            fmt.Sprintf("%f", r.Price),
				Amount:           amount.String(),
				Value:            value,
				Action:           side,
				OrderType:        orderType,
				OrderID:          r.TransactionID,
				Fee:              fee.String(),
				FeeCurrency:      feeCurrency,
				QuoteFee:         quoteFee.String(),
				QuoteFeeCurrency: quoteAssetNormalized,
				AssetDecimals:    int32(baseAsset.Decimals),
				QuoteDecimals:    int32(quoteAsset.Decimals),
				FeeDecimals:      int32(feeDecimals),
				QuoteFeeDecimals: int32(quoteFeeDecimals),
				Props:            props,
				Plugin:           g.Plugin.ID,
				PluginVersion:    g.Plugin.Version,
				Created:          timestamppb.New(time.Now().UTC()),
			}

			grpc_client.GrpcClient.SubmitTrade(context.Background(), trade)
		}

		grpc_client.GrpcClient.ShowJobProgress(context.Background(), &proto.JobProgress{
			ID:       jobId,
			Label:    fmt.Sprintf("Fetched %d trades for account \"%s\"", count, f.label),
			Progress: "-1",
		})
	}

	grpc_client.GrpcClient.AppLog(context.Background(), &proto.AppLogMsg{Level: proto.LogLevel_INFO, Message: fmt.Sprintf("[%s] Fetched %d new trades from %s.", g.Plugin.Label, count, f.label)})
	return nil
}

func (f *Fetcher) Ledger(lastFetched time.Time) ([]g.LedgerRec, error) {
	jobId := primitive.NewObjectID().Hex()
	grpc_client.GrpcClient.ShowJobProgress(context.Background(), &proto.JobProgress{
		ID:       jobId,
		Label:    fmt.Sprintf("Fetching newest transfers for account \"%s\"", f.label),
		Progress: "-1",
	})

	defer grpc_client.GrpcClient.ShowJobProgress(context.Background(), &proto.JobProgress{
		ID:       jobId,
		Progress: "100",
	})

	start := fmt.Sprintf("%d", lastFetched.Unix())
	seen := map[string]struct{}{}
	count := 0
	page := 0
	allRecs := []g.LedgerRec{}
	allSpendsAndReceives := map[string][]g.LedgerRec{}

outer:
	for {
		params := map[string]string{
			"start": start,
			"ofs":   fmt.Sprintf("%d", page),
		}

		resp, err := f.restClient.Ledgers(params)

		if err != nil {
			return allRecs, err
		}

		recs := []g.LedgerRec{}

		for lid, entry := range resp {
			if _, ok := seen[lid]; !ok {
				seen[lid] = struct{}{}
				recs = append(recs, g.LedgerRec{LedgerInfoDoc: entry, ID: lid})
			}
		}

		if len(recs) == 0 {
			break outer
		}

		page += len(recs)

		sort.Slice(recs, func(i, j int) bool {
			return recs[i].Time >= recs[j].Time
		})

		for i := range recs {
			r := recs[i]
			allRecs = append(allRecs, recs[i])
			ts := time.Unix(int64(r.Time), 0).UTC()
			asset := f.assets[r.Asset]

			switch r.Type {
			case "deposit", "withdrawal":
				count++

				transfer := &proto.Transfer{
					TxID:          r.ID,
					Ts:            timestamppb.New(ts),
					Account:       f.label,
					Fee:           g.StrToDecimal(r.Fee).Abs().String(),
					Plugin:        g.Plugin.ID,
					PluginVersion: g.Plugin.Version,
					Created:       timestamppb.New(time.Now().UTC()),
					Asset:         normalizeCurrency(r.Asset),
					AssetDecimals: int32(asset.Decimals),
					Amount:        g.StrToDecimal(r.Amount).Abs().String(),
					FeeCurrency:   normalizeCurrency(r.Asset),
					FeeDecimals:   int32(asset.Decimals),
				}

				if r.Type == "deposit" {
					transfer.Action = proto.TransferAction_DEPOSIT
					transfer.Destination = f.label
				}

				if r.Type == "withdrawal" {
					transfer.Action = proto.TransferAction_WITHDRAWAL
					transfer.Source = f.label
				}

				grpc_client.GrpcClient.SubmitTransfer(context.Background(), transfer)
			case "spend", "receive":
				if _, ok := allSpendsAndReceives[recs[i].RefID]; !ok {
					allSpendsAndReceives[recs[i].RefID] = []g.LedgerRec{}
				}

				allSpendsAndReceives[recs[i].RefID] = append(allSpendsAndReceives[recs[i].RefID], recs[i])
			}
		}

		grpc_client.GrpcClient.ShowJobProgress(context.Background(), &proto.JobProgress{
			ID:       jobId,
			Label:    fmt.Sprintf("Fetched %d transfers for account \"%s\"", count, f.label),
			Progress: "-1",
		})
	}

	// Compose trades out of "spend" and "receive" trades. These are credit card purchases.
	for refId, recs := range allSpendsAndReceives {
		spend := findRecByType("spend", recs)
		receive := findRecByType("receive", recs)

		if spend == nil || receive == nil {
			golog.Errorf("Failed to find spend and receive ledger records to compose a trade to work with (RefID = %s)", refId)
			continue
		}

		baseAssetNormalized := normalizeCurrency(receive.Asset)
		quoteAssetNormalized := normalizeCurrency(spend.Asset)
		ts := time.Unix(int64(spend.Time), 0).UTC().Add(time.Millisecond)

		baseAsset := f.assets[receive.Asset]
		quoteAsset := f.assets[spend.Asset]

		spendAmount := g.StrToDecimal(spend.Amount).Abs()
		receiveAmount := g.StrToDecimal(receive.Amount).Abs()

		props := &proto.TradeProps{
			IsMarginTrade: false,
			IsPhysical:    true,
			IsDerivative:  false,
		}

		trade := &proto.Trade{
			TxID:             refId,
			Ts:               timestamppb.New(ts),
			Account:          f.label,
			Ticker:           fmt.Sprintf("%s/%s", baseAssetNormalized, quoteAssetNormalized),
			Asset:            baseAssetNormalized,
			Quote:            quoteAssetNormalized,
			Price:            spendAmount.Div(receiveAmount).String(),
			Amount:           receiveAmount.String(),
			Value:            spendAmount.String(),
			OrderType:        proto.OrderType_TAKER,
			OrderID:          spend.ID,
			Fee:              g.StrToDecimal(receive.Fee).Abs().String(),
			FeeCurrency:      baseAssetNormalized,
			QuoteFee:         g.StrToDecimal(spend.Fee).Abs().String(),
			QuoteFeeCurrency: quoteAssetNormalized,
			AssetDecimals:    int32(baseAsset.Decimals),
			QuoteDecimals:    int32(quoteAsset.Decimals),
			FeeDecimals:      int32(baseAsset.Decimals),
			QuoteFeeDecimals: int32(quoteAsset.Decimals),
			Props:            props,
			Plugin:           g.Plugin.ID,
			PluginVersion:    g.Plugin.Version,
			Created:          timestamppb.Now(),
			Comment:          "Credit card purchase",
		}

		grpc_client.GrpcClient.SubmitTrade(context.Background(), trade)
	}

	grpc_client.GrpcClient.AppLog(context.Background(), &proto.AppLogMsg{Level: proto.LogLevel_INFO, Message: fmt.Sprintf("[%s] Fetched %d new transfers from %s.", g.Plugin.Label, count, f.label)})
	return allRecs, nil
}

func findRecByType(t string, recs []g.LedgerRec) *g.LedgerRec {
	for i, r := range recs {
		if r.Type == t {
			return &recs[i]
		}
	}

	return nil
}
