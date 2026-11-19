// Package feed reads and normalizes public Bybit V5 market data.
package feed

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	defaultRESTURL = "https://api.bybit.com"
	defaultWSURL   = "wss://stream.bybit.com"
	requestTimeout = 15 * time.Second
)

// ErrInvalidSymbols identifies an invalid explicitly configured symbol list.
var ErrInvalidSymbols = errors.New("invalid Bybit symbols")

// Client reads Bybit's public instrument directory and live market feeds.
type Client struct {
	restURL string
	wsURL   string
	http    *http.Client
}

// NewClient uses Bybit mainnet for any empty base URL.
func NewClient(restURL, wsBaseURL string) *Client {
	return &Client{
		restURL: strings.TrimRight(cmp.Or(restURL, defaultRESTURL), "/"),
		wsURL:   strings.TrimRight(cmp.Or(wsBaseURL, defaultWSURL), "/"),
		http:    &http.Client{Timeout: requestTimeout},
	}
}

type instrumentInfo struct {
	Symbol       string `json:"symbol"`
	QuoteCoin    string `json:"quoteCoin"`
	SettleCoin   string `json:"settleCoin"`
	Status       string `json:"status"`
	ContractType string `json:"contractType"`
	IsPreListing bool   `json:"isPreListing"`
	Price        struct {
		Tick string `json:"tickSize"`
		Min  string `json:"minPrice"`
		Max  string `json:"maxPrice"`
	} `json:"priceFilter"`
	Lot struct {
		BasePrecision string `json:"basePrecision"`
		Step          string `json:"qtyStep"`
		Min           string `json:"minOrderQty"`
		Max           string `json:"maxOrderQty"`
		SpotMaxMarket string `json:"maxMarketOrderQty"`
		PerpMaxMarket string `json:"maxMktOrderQty"`
		SpotMinAmount string `json:"minOrderAmt"`
		PerpMinAmount string `json:"minNotionalValue"`
	} `json:"lotSizeFilter"`
}

type marketResult[T any] struct {
	Category model.Market `json:"category"`
	List     []T          `json:"list"`
	Next     string       `json:"nextPageCursor"`
}

type ticker struct {
	Symbol   string `json:"symbol"`
	Turnover string `json:"turnover24h"`
}

type candidate struct {
	info     instrumentInfo
	turnover float64
}

func supportedMarket(market model.Market) bool {
	return market == model.Spot || market == model.Linear
}

func validTopicSymbol(symbol string) bool {
	return symbol != "" && !strings.ContainsAny(symbol, ". ,\t\r\n")
}

// Instruments selects active USDT instruments by explicit symbols or by 24-hour turnover.
func (c *Client) Instruments(ctx context.Context, market model.Market, symbols []string, limit int) ([]model.Instrument, error) {
	if !supportedMarket(market) {
		return nil, fmt.Errorf("unsupported Bybit market %q", market)
	}
	if len(symbols) == 0 && limit <= 0 {
		return nil, errors.New("instrument limit must be positive")
	}

	names, err := normalizeSymbols(symbols)
	if err != nil {
		return nil, err
	}

	directory, err := c.directory(ctx, market)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if _, ok := directory[name]; !ok {
			return nil, fmt.Errorf("%w: %s:%s is outside the active USDT market", ErrInvalidSymbols, market, name)
		}
	}

	var tickers marketResult[ticker]
	query := url.Values{"category": {string(market)}}
	if err := c.get(ctx, "/v5/market/tickers", query, &tickers); err != nil {
		return nil, err
	}
	if tickers.Category != market {
		return nil, fmt.Errorf("tickers returned category %q for %q", tickers.Category, market)
	}

	selected, err := selectInstruments(directory, tickers.List, market, names, limit)
	if err != nil {
		return nil, err
	}

	result := make([]model.Instrument, 0, len(selected))
	for _, item := range selected {
		instrument, err := item.instrument(market)
		if err != nil {
			return nil, fmt.Errorf("instrument %s:%s: %w", market, item.info.Symbol, err)
		}
		result = append(result, instrument)
	}

	return result, nil
}

func normalizeSymbols(input []string) ([]string, error) {
	result := make([]string, 0, len(input))
	seen := make(map[string]bool, len(input))
	for _, value := range input {
		name := strings.ToUpper(strings.TrimSpace(value))
		if !validTopicSymbol(name) || seen[name] {
			return nil, fmt.Errorf("%w: %q is empty, malformed or duplicated", ErrInvalidSymbols, value)
		}
		seen[name] = true
		result = append(result, name)
	}

	return result, nil
}

func (c *Client) directory(ctx context.Context, market model.Market) (map[string]instrumentInfo, error) {
	query := url.Values{"category": {string(market)}, "status": {"Trading"}}
	if market == model.Linear {
		query.Set("limit", "1000")
	}

	result := make(map[string]instrumentInfo)
	seen := make(map[string]bool)
	for {
		var page marketResult[instrumentInfo]
		if err := c.get(ctx, "/v5/market/instruments-info", query, &page); err != nil {
			return nil, err
		}
		if page.Category != market {
			return nil, fmt.Errorf("directory returned category %q for %q", page.Category, market)
		}
		if page.List == nil {
			return nil, fmt.Errorf("instrument list omitted by Bybit %s directory", market)
		}
		for _, info := range page.List {
			if info.activeUSDT(market) && validTopicSymbol(info.Symbol) {
				result[info.Symbol] = info
			}
		}

		if market == model.Spot || page.Next == "" {
			if len(result) == 0 {
				return nil, fmt.Errorf("no active USDT instruments in Bybit %s directory", market)
			}

			return result, nil
		}
		if seen[page.Next] {
			return nil, fmt.Errorf("repeated pagination cursor in Bybit %s directory", market)
		}
		seen[page.Next] = true
		query.Set("cursor", page.Next)
	}
}

func (i instrumentInfo) activeUSDT(market model.Market) bool {
	if i.Status != "Trading" || i.QuoteCoin != "USDT" {
		return false
	}

	return market == model.Spot || (i.ContractType == "LinearPerpetual" && i.SettleCoin == "USDT" && !i.IsPreListing)
}

func selectInstruments(directory map[string]instrumentInfo, tickers []ticker, market model.Market, names []string, limit int) ([]candidate, error) {
	available := make(map[string]candidate, len(directory))
	for _, ticker := range tickers {
		info, ok := directory[ticker.Symbol]
		if !ok {
			continue
		}
		turnover, err := nonnegative(ticker.Turnover)
		if err != nil {
			continue
		}
		available[ticker.Symbol] = candidate{info: info, turnover: turnover}
	}

	if len(names) > 0 {
		result := make([]candidate, 0, len(names))
		for _, name := range names {
			item, ok := available[name]
			if !ok {
				return nil, fmt.Errorf("missing Bybit turnover for %s:%s", market, name)
			}
			result = append(result, item)
		}

		return result, nil
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no eligible Bybit %s instruments with turnover", market)
	}

	result := make([]candidate, 0, len(available))
	for _, item := range available {
		result = append(result, item)
	}
	slices.SortFunc(result, func(left, right candidate) int {
		if order := cmp.Compare(right.turnover, left.turnover); order != 0 {
			return order
		}

		return cmp.Compare(left.info.Symbol, right.info.Symbol)
	})

	return result[:min(limit, len(result))], nil
}

func (c candidate) instrument(market model.Market) (model.Instrument, error) {
	instrument := model.Instrument{
		Key:          model.Key{Market: market, Symbol: c.info.Symbol},
		TickSizeText: c.info.Price.Tick,
		Turnover:     c.turnover,
	}
	if _, err := parseDecimal(c.info.Price.Tick); err != nil {
		return model.Instrument{}, fmt.Errorf("tick size: %w", err)
	}

	lot := c.info.Lot
	qtyStep := lot.Step
	maxQty := cmp.Or(lot.PerpMaxMarket, lot.Max)
	minAmount := lot.PerpMinAmount
	if market == model.Spot {
		qtyStep = lot.BasePrecision
		maxQty = cmp.Or(lot.SpotMaxMarket, lot.Max)
		minAmount = lot.SpotMinAmount
	}

	type instrumentField struct {
		value string
		name  string
		dest  *float64
	}
	fields := []instrumentField{
		{value: c.info.Price.Tick, name: "tick size", dest: &instrument.TickSize},
		{value: qtyStep, name: "quantity step", dest: &instrument.QtyStep},
		{value: maxQty, name: "maximum quantity", dest: &instrument.MaxQty},
		{value: minAmount, name: "minimum order amount", dest: &instrument.MinNotional},
	}
	if market == model.Linear {
		fields = append(fields,
			instrumentField{value: lot.Min, name: "minimum quantity", dest: &instrument.MinQty},
			instrumentField{value: c.info.Price.Min, name: "minimum price", dest: &instrument.MinPrice},
			instrumentField{value: c.info.Price.Max, name: "maximum price", dest: &instrument.MaxPrice},
		)
	}
	for _, field := range fields {
		value, err := nonnegative(field.value)
		if err != nil || value <= 0 {
			return model.Instrument{}, fmt.Errorf("invalid %s %q", field.name, field.value)
		}
		*field.dest = value
	}
	if instrument.MinQty > instrument.MaxQty || instrument.MinPrice > instrument.MaxPrice {
		return model.Instrument{}, errors.New("invalid instrument bounds")
	}

	return instrument, nil
}

func nonnegative(raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, fmt.Errorf("invalid nonnegative number %q", raw)
	}

	return value, nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.restURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("create Bybit request for %s: %w", path, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request Bybit %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for Bybit %s", resp.StatusCode, path)
	}

	var envelope struct {
		Code    *int            `json:"retCode"`
		Message string          `json:"retMsg"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&envelope); err != nil {
		return fmt.Errorf("decode Bybit %s: %w", path, err)
	}
	if envelope.Code == nil {
		return fmt.Errorf("missing retCode in Bybit %s response", path)
	}
	if *envelope.Code != 0 {
		return fmt.Errorf("retCode %d from Bybit %s: %s", *envelope.Code, path, envelope.Message)
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode Bybit %s result: %w", path, err)
	}

	return nil
}
