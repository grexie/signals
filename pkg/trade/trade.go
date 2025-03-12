package trade

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/grexie/signals/pkg/model"
)

var (
	API_KEY        = func() string { return os.Getenv("OKX_API_KEY") }
	API_SECRET     = func() string { return os.Getenv("OKX_API_SECRET") }
	API_PASSPHRASE = func() string { return os.Getenv("OKX_API_PASSPHRASE") }
	OKX_BASE_URL   = func() string { return os.Getenv("OKX_BASE_URL") }
)

type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

type PositionSide string

const (
	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"
)

type TickerResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Last  string `json:"last"`
		AskPx string `json:"askPx"`
		BidPx string `json:"bidPx"`
	} `json:"data"`
}

func GetCurrentPrice(instrument string) (float64, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.okx.com/api/v5/market/ticker?instId=%s", instrument))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var ticker TickerResponse
	err = json.Unmarshal(body, &ticker)
	if err != nil {
		return 0, err
	}

	if len(ticker.Data) == 0 {
		return 0, fmt.Errorf("no data received from API when getting mark price")
	}

	// Convert string to float
	var lastPrice float64
	fmt.Sscanf(ticker.Data[0].Last, "%f", &lastPrice)

	return lastPrice, nil
}

type OrderDetails struct {
	Instrument string `json:"instId"`
	OrderID    string `json:"ordId"`
}

type BalancesResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Details []struct {
			Currency string `json:"ccy"`
			Equity   string `json:"eq"`
		} `json:"details"`
	} `json:"data"`
}

func GetEquity(ctx context.Context) (float64, error) {
	client := resty.New()

	url := OKX_BASE_URL() + "/api/v5/account/balance"

	now := time.Now()
	timestamp := now.UTC().Format("2006-01-02T15:04:05.999Z")
	signature := generateSignature(timestamp, "GET", "/api/v5/account/balance", nil)

	resp, err := client.R().
		SetHeaders(map[string]string{
			"OK-ACCESS-KEY":        API_KEY(),
			"OK-ACCESS-SIGN":       signature,
			"OK-ACCESS-TIMESTAMP":  timestamp,
			"OK-ACCESS-PASSPHRASE": API_PASSPHRASE(),
			"Content-Type":         "application/json",
		}).
		Get(url)
	if err != nil {
		return 0, err
	}

	var res BalancesResponse
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return 0, err
	}

	for _, details := range res.Data[0].Details {
		if details.Currency == "USDT" {
			// Convert string to float
			var equity float64
			fmt.Sscanf(details.Equity, "%f", &equity)
			return equity, nil
		}
	}

	return 0, nil
}

type OrderResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClientOrderID  string `json:"clOrdId"`
		OrderID        string `json:"ordId"`
		Tag            string `json:"tag"`
		Timestamp      string `json:"ts"`
		SuccessCode    string `json:"sCode"`
		SuccessMessage string `json:"sMsg"`
	} `json:"data"`
	InTime  string `json:"inTime"`
	OutTime string `json:"outTime"`
}

func getContractMultiplier(instId string) (float64, float64, error) {
	path := fmt.Sprintf("%s/api/v5/public/instruments?instType=SWAP&instId=%s", OKX_BASE_URL(), instId)
	response, err := http.Get(path)
	if err != nil {
		return 0, 0, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, 0, err
	}

	var result map[string]any
	json.Unmarshal(body, &result)

	if result["code"] != "0" {
		return 0, 0, fmt.Errorf("failed to fetch instrument data: %s", result["msg"])
	}

	data := result["data"].([]any)[0].(map[string]any)
	contractMultiplier, _ := strconv.ParseFloat(data["ctVal"].(string), 64)
	maxSz, _ := strconv.ParseFloat(data["maxMktSz"].(string), 64) // OKX max allowed position size

	return contractMultiplier, maxSz, nil
}

func PlaceOrder(ctx context.Context, instrument string, isLong bool, usdt float64, takeProfit float64, stopLoss float64, leverage float64) (*OrderDetails, error) {
	client := resty.New()

	tdMode := "isolated"
	lever := fmt.Sprintf("%f", leverage)

	side := OrderSideBuy
	posSide := PositionSideLong
	if !isLong {
		posSide = PositionSideShort
		side = OrderSideSell
	}

	entryPrice, err := GetCurrentPrice(instrument)
	if err != nil {
		return nil, err
	}

	// Get contract multiplier and max size
	contractMultiplier, maxSz, err := getContractMultiplier(instrument)
	if err != nil {
		return nil, err
	}

	tp := entryPrice * (1 + takeProfit/leverage)
	sl := entryPrice * (1 - stopLoss/leverage)
	if !isLong {
		tp = entryPrice * (1 - takeProfit/leverage)
		sl = entryPrice * (1 + stopLoss/leverage)
	}

	tpTriggerPx := fmt.Sprintf("%.6f", tp)
	tpOrdPx := "-1"
	slTriggerPx := fmt.Sprintf("%.6f", sl)
	slOrdPx := "-1"

	url := OKX_BASE_URL() + "/api/v5/trade/order"

	quantity := (leverage * (usdt * (1 - leverage*model.Commission))) / (entryPrice * contractMultiplier)

	if quantity > maxSz {
		quantity = maxSz
	}

	sz := fmt.Sprintf("%0.2f", quantity)

	body := map[string]string{
		"instId":      instrument,
		"tdMode":      tdMode,
		"side":        string(side),
		"posSide":     string(posSide),
		"ordType":     "market",
		"lever":       lever,
		"sz":          sz,
		"tpTriggerPx": tpTriggerPx,
		"tpOrdPx":     tpOrdPx,
		"slTriggerPx": slTriggerPx,
		"slOrdPx":     slOrdPx,
	}

	now := time.Now()
	timestamp := now.UTC().Format("2006-01-02T15:04:05.999Z")
	signature := generateSignature(timestamp, "POST", "/api/v5/trade/order", body)

	log.Printf("placing market order %s: %s at price %0.06f (TP %s / SL %s)", instrument, sz, entryPrice, tpTriggerPx, slTriggerPx)

	resp, err := client.R().
		SetHeaders(map[string]string{
			"OK-ACCESS-KEY":        API_KEY(),
			"OK-ACCESS-SIGN":       signature,
			"OK-ACCESS-TIMESTAMP":  timestamp,
			"OK-ACCESS-PASSPHRASE": API_PASSPHRASE(),
			"Content-Type":         "application/json",
		}).
		SetBody(body).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}

	var order OrderResponse
	if err := json.Unmarshal(resp.Body(), &order); err != nil {
		return nil, err
	}

	if order.Code != "0" {
		return nil, fmt.Errorf("failed to place order: %s", order.Msg)
	}

	return &OrderDetails{
		Instrument: instrument,
		OrderID:    order.Data[0].OrderID,
	}, nil
}

// 🔐 Generate OKX API Signature
func generateSignature(timestamp, method, requestPath string, body map[string]string) string {
	var payload string

	// Convert request body to JSON if it exists
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		payload = fmt.Sprintf("%s%s%s%s", timestamp, method, requestPath, string(jsonBody))
	} else {
		payload = fmt.Sprintf("%s%s%s", timestamp, method, requestPath)
	}

	// HMAC-SHA256 Signature
	h := hmac.New(sha256.New, []byte(API_SECRET()))
	h.Write([]byte(payload))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

type AccountPositionsResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		InstrumentID  string `json:"instId"`
		PositionSide  string `json:"posSide"`
		Leverage      string `json:"lever"`
		Position      string `json:"pos"`
		AveragePrice  string `json:"avgPx"`
		UnrealisedPnL string `json:"upl"`
		Margin        string `json:"margin"`
	} `json:"data"`
}

func GetPositions(ctx context.Context) (*AccountPositionsResponse, error) {
	client := resty.New()

	url := fmt.Sprintf("%s/api/v5/account/positions", OKX_BASE_URL())

	now := time.Now()
	timestamp := now.UTC().Format("2006-01-02T15:04:05.999Z")
	signature := generateSignature(timestamp, "GET", "/api/v5/account/positions", nil)

	resp, err := client.R().
		SetContext(ctx).
		SetHeaders(map[string]string{
			"OK-ACCESS-KEY":        API_KEY(),
			"OK-ACCESS-SIGN":       signature,
			"OK-ACCESS-TIMESTAMP":  timestamp,
			"OK-ACCESS-PASSPHRASE": API_PASSPHRASE(),
			"Content-Type":         "application/json",
		}).
		Get(url)
	if err != nil {
		return nil, err
	}

	var res AccountPositionsResponse
	err = json.Unmarshal(resp.Body(), &res)
	if err != nil {
		return nil, err
	}

	if res.Code != "0" {
		return nil, fmt.Errorf("error retrieving positions: %v", res.Msg)
	}

	return &res, nil
}

func CheckPositions(ctx context.Context, instrument string) (bool, error) {
	if positions, err := GetPositions(ctx); err != nil {
		return false, err
	} else {
		for _, p := range positions.Data {
			if p.InstrumentID == instrument {
				return true, nil
			}
		}
	}
	return false, nil
}
