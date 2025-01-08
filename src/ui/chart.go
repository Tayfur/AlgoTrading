package ui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"trading-analyzer/src/analyzer"
	"trading-analyzer/src/exchange"
	"trading-analyzer/src/models"
	"trading-analyzer/src/utils"

	"github.com/gorilla/websocket"
)

type ChartUI struct {
	port   int
	client *exchange.Client
}

func NewChartUI(client *exchange.Client) *ChartUI {
	return &ChartUI{
		port:   3000,
		client: client,
	}
}

func (c *ChartUI) Run() {
	// Add CORS headers
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Apply middleware
	handler := corsMiddleware(http.DefaultServeMux)

	// Setup routes
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("src/ui/static"))))
	http.HandleFunc("/", c.serveChart)
	http.HandleFunc("/ws", c.handleWebSocket)

	serverAddr := fmt.Sprintf(":%d", c.port)
	log.Printf("Starting chart server at http://localhost%s\n", serverAddr)

	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		log.Fatal("Server error:", err)
	}
}

func (c *ChartUI) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Create channels for this connection
	candleChan := make(chan models.Candle)
	signalChan := make(chan analyzer.Signal)

	// Start data collectors
	go c.collectData(conn, candleChan, signalChan)

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg struct {
			Type      string `json:"type"`
			Timeframe string `json:"timeframe"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "subscribe" && msg.Timeframe != "" {
			// Get historical data
			candles, err := c.client.GetCandles("BTCUSDT", msg.Timeframe, 1000)
			if err != nil {
				log.Printf("Error fetching historical candles: %v", err)
				continue
			}

			// Send historical data
			if err := conn.WriteJSON(map[string]interface{}{
				"type":    "history",
				"candles": candles,
			}); err != nil {
				log.Printf("Error sending historical data: %v", err)
				break
			}
		}
	}
}

func (c *ChartUI) serveChart(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "src/ui/templates/chart.html")
}

func (c *ChartUI) collectCandles(candleChan chan models.Candle) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		candles, err := c.client.GetCandles("BTCUSDT", "1m", 1)
		if err != nil {
			log.Printf("Error fetching candles: %v", err)
			continue
		}

		if len(candles) > 0 {
			candleChan <- candles[0]
		}
	}
}

func calculateRisk(candles []models.Candle) float64 {
	prices := make([]float64, len(candles))
	for i, c := range candles {
		prices[i] = c.Close
	}
	rsi := utils.RSI(prices, 14)
	if len(rsi) > 0 {
		return (100 - rsi[len(rsi)-1]) / 100 * 10
	}
	return 5.0
}

func (c *ChartUI) collectSignals(signalChan chan analyzer.Signal) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	timeframes := []string{"5m", "30m", "1h"}

	for range ticker.C {
		for _, tf := range timeframes {
			candles, err := c.client.GetCandles("BTCUSDT", tf, 100)
			if err != nil {
				log.Printf("Error fetching %s candles: %v", tf, err)
				continue
			}

			patterns := analyzer.AnalyzePatterns(candles)
			prediction := analyzer.PredictPrice(candles, tf)
			risk := calculateRisk(candles)

			signal := analyzer.Signal{
				Type:      prediction.Direction,
				Risk:      risk,
				Price:     candles[len(candles)-1].Close,
				Strength:  len(patterns),
				TimeFrame: tf,
			}

			signalChan <- signal
		}
	}
}

func (c *ChartUI) collectData(conn *websocket.Conn, candleChan chan models.Candle, signalChan chan analyzer.Signal) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		candles, err := c.client.GetCandles("BTCUSDT", "1m", 1)
		if err != nil {
			log.Printf("Error fetching latest candle: %v", err)
			continue
		}

		if len(candles) > 0 {
			if err := conn.WriteJSON(map[string]interface{}{
				"type":   "candle",
				"candle": candles[0],
			}); err != nil {
				return
			}
		}

		// Send signals every minute
		if time.Now().Second() == 0 {
			if len(candles) > 0 {
				prediction := analyzer.PredictPrice(candles, "1m")
				signal := analyzer.Signal{
					Type:      prediction.Direction,
					Price:     candles[0].Close,
					Risk:      calculateRisk(candles),
					TimeFrame: "1m",
				}

				if err := conn.WriteJSON(map[string]interface{}{
					"type":   "signal",
					"signal": signal,
				}); err != nil {
					return
				}
			}
		}
	}
}
