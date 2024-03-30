package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	krakenapi "github.com/beldur/kraken-go-api-client"
	g "github.com/f-taxes/kraken_import/global"
	"github.com/f-taxes/kraken_import/grpc_client"
	"github.com/f-taxes/kraken_import/proto"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/ratelimit"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Fetcher struct {
	label      string
	restClient *krakenapi.KrakenAPI
	assets     map[string]AssetInfo
	pairs      map[string]PairInfo
}

func New(label, key, secret string) (*Fetcher, error) {
	client := krakenapi.New(key, secret)

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

func (f *Fetcher) Trades(lastFetched time.Time) error {
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
		limiter.Take()

		params := map[string]string{
			"ofs": fmt.Sprintf("%d", page),
		}

		resp, err := f.restClient.TradesHistory(start, 0, params)

		if err != nil {
			return err
		}

		recs := []g.TradeRec{}

		for lid, entry := range resp.Trades {
			if _, ok := seen[lid]; !ok {
				seen[lid] = struct{}{}
				recs = append(recs, g.TradeRec{TradeHistoryInfo: entry, ID: lid})
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

			trade := &proto.Trade{
				TxID:          r.ID,
				Ts:            timestamppb.New(ts),
				Account:       f.label,
				Ticker:        pair.Wsname,
				Quote:         normalizeCurrency(quoteAsset.Altname),
				Asset:         normalizeCurrency(baseAsset.Altname),
				AssetType:     proto.AssetType_SPOT_MARGIN,
				Price:         fmt.Sprintf("%f", r.Price),
				Amount:        fmt.Sprintf("%f", r.Volume),
				Value:         fmt.Sprintf("%f", r.Cost),
				Action:        side,
				OrderType:     orderType,
				OrderID:       r.TransactionID,
				Fee:           fmt.Sprintf("%f", r.Fee),
				FeeCurrency:   normalizeCurrency(quoteAsset.Altname),
				Plugin:        g.Plugin.ID,
				PluginVersion: g.Plugin.Version,
				Created:       timestamppb.New(time.Now().UTC()),
			}

			grpc_client.GrpcClient.SubmitTrade(context.Background(), trade)
		}
	}

	grpc_client.GrpcClient.AppLog(context.Background(), &proto.AppLogMsg{Level: proto.LogLevel_INFO, Message: fmt.Sprintf("[%s] Fetched %d new trades from %s.", g.Plugin.Label, count, f.label)})
	return nil
}

func (f *Fetcher) Ledger(lastFetched time.Time) error {
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

outer:
	for {
		limiter.Take()

		params := map[string]string{
			"start": start,
			"ofs":   fmt.Sprintf("%d", page),
		}

		resp, err := f.restClient.Ledgers(params)

		if err != nil {
			return err
		}

		recs := []g.LedgerRec{}

		for lid, entry := range resp.Ledger {
			if _, ok := seen[lid]; !ok {
				seen[lid] = struct{}{}
				recs = append(recs, g.LedgerRec{LedgerInfo: entry, ID: lid})
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

			switch r.Type {
			case "deposit", "withdrawal":
				count++

				transfer := &proto.Transfer{
					TxID:          r.ID,
					Ts:            timestamppb.New(ts),
					Account:       f.label,
					Fee:           g.StrToDecimal(r.Fee.Text('f', 8)).Abs().String(),
					Plugin:        g.Plugin.ID,
					PluginVersion: g.Plugin.Version,
					Created:       timestamppb.New(time.Now().UTC()),
					Asset:         normalizeCurrency(r.Asset),
					Amount:        g.StrToDecimal(r.Amount.Text('f', 8)).Abs().String(),
					FeeCurrency:   normalizeCurrency(r.Asset),
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
			}
		}
	}

	grpc_client.GrpcClient.AppLog(context.Background(), &proto.AppLogMsg{Level: proto.LogLevel_INFO, Message: fmt.Sprintf("[%s] Fetched %d new transfers from %s.", g.Plugin.Label, count, f.label)})
	return nil
}
