package exchange

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"trading-analyzer/src/models"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new exchange client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *Client) GetCandles(symbol, interval string, limit int) ([]models.Candle, error) {
	url := fmt.Sprintf("%s/klines?symbol=%s&interval=%s&limit=%d",
		c.BaseURL, symbol, interval, limit)

	log.Printf("Fetching candles: %s", url)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("API Response Status: %s", resp.Status)

	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %v", err)
	}

	log.Printf("Received %d candles from API", len(rawData))

	candles := make([]models.Candle, 0, len(rawData))
	for _, raw := range rawData {
		if len(raw) < 6 {
			continue
		}

		candle := models.Candle{
			Time:   int64(raw[0].(float64)), // Open time
			Open:   parseFloat(raw[1]),      // Open
			High:   parseFloat(raw[2]),      // High
			Low:    parseFloat(raw[3]),      // Low
			Close:  parseFloat(raw[4]),      // Close
			Volume: parseFloat(raw[5]),      // Volume
		}
		candles = append(candles, candle)
	}

	log.Printf("Processed %d valid candles", len(candles))
	return candles, nil
}

// GetCurrentPrice fetches the latest price for a symbol
func (c *Client) GetCurrentPrice(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/ticker/price?symbol=%s", c.BaseURL, symbol)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("JSON decode failed: %v", err)
	}

	return parseStringToFloat(result.Price), nil
}

func convertToCandle(raw []interface{}) (models.Candle, error) {
	if len(raw) < 6 {
		return models.Candle{}, fmt.Errorf("invalid data length")
	}

	// Parse timestamp
	timestamp := int64(raw[0].(float64))

	// Parse OHLCV
	open, _ := raw[1].(string)
	high, _ := raw[2].(string)
	low, _ := raw[3].(string)
	close, _ := raw[4].(string)
	volume, _ := raw[5].(string)

	return models.Candle{
		Time:   timestamp,
		Open:   parseStringToFloat(open),
		High:   parseStringToFloat(high),
		Low:    parseStringToFloat(low),
		Close:  parseStringToFloat(close),
		Volume: parseStringToFloat(volume),
	}, nil
}

func parseStringToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseFloat(v interface{}) float64 {
	switch t := v.(type) {
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	case float64:
		return t
	default:
		return 0
	}
}
