package fetcher

type AssetInfo struct {
	Aclass          string `json:"aclass"`
	Altname         string `json:"altname"`
	Decimals        int    `json:"decimals"`
	DisplayDecimals int    `json:"display_decimals"`
	Status          string `json:"status"`
}

type AssetsResponse struct {
	Error  []any                `json:"error"`
	Result map[string]AssetInfo `json:"result"`
}

type PairResponse struct {
	Error  []any               `json:"error"`
	Result map[string]PairInfo `json:"result"`
}

type PairInfo struct {
	Altname           string      `json:"altname"`
	Wsname            string      `json:"wsname"`
	AclassBase        string      `json:"aclass_base"`
	Base              string      `json:"base"`
	AclassQuote       string      `json:"aclass_quote"`
	Quote             string      `json:"quote"`
	Lot               string      `json:"lot"`
	CostDecimals      int         `json:"cost_decimals"`
	PairDecimals      int         `json:"pair_decimals"`
	LotDecimals       int         `json:"lot_decimals"`
	LotMultiplier     int         `json:"lot_multiplier"`
	LeverageBuy       []any       `json:"leverage_buy"`
	LeverageSell      []any       `json:"leverage_sell"`
	Fees              [][]float64 `json:"fees"`
	FeesMaker         [][]float64 `json:"fees_maker"`
	FeeVolumeCurrency string      `json:"fee_volume_currency"`
	MarginCall        int         `json:"margin_call"`
	MarginStop        int         `json:"margin_stop"`
	Ordermin          string      `json:"ordermin"`
	Costmin           string      `json:"costmin"`
	TickSize          string      `json:"tick_size"`
	Status            string      `json:"status"`
}
