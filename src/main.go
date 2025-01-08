package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
	"trading-analyzer/src/analyzer"
	"trading-analyzer/src/exchange"
	"trading-analyzer/src/models"
	"trading-analyzer/src/ui"
	"trading-analyzer/src/utils"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("Initializing Trading Analyzer Terminal...")

	client := exchange.NewClient("https://api.binance.com/api/v3")
	symbol := "BTCUSDT"
	timeframes := []string{"5m", "30m", "1h"}

	// Create UI with client
	chartUI := ui.NewChartUI(client)

	// Create channels for price updates and pattern analysis
	priceChan := make(chan float64)
	patternChan := make(chan []string)

	// Start real-time price monitoring
	go monitorPrice(client, symbol, priceChan)

	// Start pattern analysis
	go analyzePatterns(client, symbol, timeframes, patternChan)

	// Run UI (this will block)
	chartUI.Run()
}

func monitorPrice(client *exchange.Client, symbol string, priceChan chan float64) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		price, err := client.GetCurrentPrice(symbol)
		if err != nil {
			log.Printf("Error fetching price: %v\n", err)
			continue
		}
		priceChan <- price
	}
}

func AnalyzePatterns(candles []models.Candle) []string {
	var patterns []string

	// Need at least 2 candles for pattern analysis
	if len(candles) < 2 {
		return patterns
	}

	last := candles[len(candles)-1]
	prev := candles[len(candles)-2]

	// Bullish patterns
	if last.Close > prev.Close {
		if last.Close > prev.Open && last.Open < prev.Close {
			patterns = append(patterns, "BULLISH ENGULFING")
		}
		if last.Close-last.Open > 2*(prev.Close-prev.Open) {
			patterns = append(patterns, "STRONG BULLISH MOMENTUM")
		}
	}

	// Bearish patterns
	if last.Close < prev.Close {
		if last.Close < prev.Open && last.Open > prev.Close {
			patterns = append(patterns, "BEARISH ENGULFING")
		}
		if last.Open-last.Close > 2*(prev.Open-prev.Close) {
			patterns = append(patterns, "STRONG BEARISH MOMENTUM")
		}
	}

	return patterns
}

func getHistoricalCandles(client *exchange.Client, symbol, timeframe string) ([]models.Candle, error) {
	// Fetch last year's worth of candles
	return client.GetCandles(symbol, timeframe, 365*24*60/timeframeToMinutes(timeframe))
}

func findSimilarPatterns(current, historical []models.Candle) []string {
	var patterns []string
	currentPattern := analyzer.AnalyzePatterns(current[len(current)-5:])

	// Look for similar patterns in historical data
	for i := 0; i < len(historical)-5; i++ {
		window := historical[i : i+5]
		if patternMatches(currentPattern, analyzer.AnalyzePatterns(window)) {
			patterns = append(patterns, "match_found")
		}
	}
	return patterns
}

func patternMatches(pattern1, pattern2 []string) bool {
	if len(pattern1) != len(pattern2) {
		return false
	}
	for i := range pattern1 {
		if pattern1[i] != pattern2[i] {
			return false
		}
	}
	return true
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

func timeframeToMinutes(tf string) int {
	switch tf {
	case "5m":
		return 5
	case "30m":
		return 30
	case "1h":
		return 60
	default:
		return 60
	}
}

func printPatternAnalysis(patterns []string) {
	fmt.Println("\nPattern Analysis Results:")
	for _, p := range patterns {
		fmt.Printf("- %s\n", p)
	}
}

func analyzePatterns(client *exchange.Client, symbol string, timeframes []string, patternChan chan []string) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		for _, tf := range timeframes {
			candles, err := client.GetCandles(symbol, tf, 100)
			if err != nil {
				log.Printf("Error fetching %s candles: %v\n", tf, err)
				continue
			}

			// Get current pattern
			currentPattern := AnalyzePatterns(candles[len(candles)-5:])

			// Get price prediction
			prediction := analyzer.PredictPrice(candles, tf)

			// Calculate risk
			risk := calculateRisk(candles)

			signals := []string{
				fmt.Sprintf("=== %s Timeframe Analysis ===", tf),
				fmt.Sprintf("Pattern: %s", strings.Join(currentPattern, ", ")),
				fmt.Sprintf("Current Price: $%.2f", candles[len(candles)-1].Close),
				fmt.Sprintf("Predicted Price (%s): $%.2f", prediction.TimeFrame, prediction.PredictedPrice),
				fmt.Sprintf("Prediction Confidence: %.1f%%", prediction.Confidence),
				fmt.Sprintf("Predicted Direction: %s", prediction.Direction),
				fmt.Sprintf("Risk Level: %.2f%%", risk),
			}

			// Find support/resistance levels
			levels := analyzer.FindSupportResistance(candles)

			// Add S/R levels to signals
			signals = append(signals, "\nSupport/Resistance Levels:")
			for _, level := range levels {
				signals = append(signals, fmt.Sprintf("%s at $%.2f (Strength: %.1f%%, Touches: %d)",
					level.Type, level.Price, level.Strength, level.TouchCount))
			}

			patternChan <- signals
		}
	}
}

func formatAnalysis(timeframe string, signals []analyzer.Signal) []string {
	var analysis []string
	analysis = append(analysis, fmt.Sprintf("=== %s Timeframe Analysis ===", timeframe))

	if len(signals) == 0 {
		analysis = append(analysis, "No strong signals detected")
		return analysis
	}

	for _, signal := range signals {
		analysis = append(analysis, fmt.Sprintf("Signal: %s | Strength: %d | Risk: %.2f%% | Entry Price: $%.2f",
			signal.Type, signal.Strength, signal.Risk, signal.Price))
	}

	return analysis
}

func startWebSocket(client *exchange.Client) {
	http.HandleFunc("/ws", handleWebSocket)

	// Serve the HTML template
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("src/ui/templates/chart.html"))
		tmpl.Execute(w, nil)
	})

	// Create channels for data distribution
	candleChan := make(chan models.Candle)
	signalChan := make(chan analyzer.Signal)

	// Start data collection goroutines
	go collectCandles(client, candleChan)
	go collectSignals(client, signalChan)

	// Start the HTTP server
	log.Printf("Starting WebSocket server on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal("WebSocket server error:", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all connections in development
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Handle the WebSocket connection
	for {
		// Keep connection alive and handle incoming messages
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
	}
}

func collectCandles(client *exchange.Client, candleChan chan models.Candle) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		candles, err := client.GetCandles("BTCUSDT", "1m", 1)
		if err != nil {
			log.Printf("Error fetching candles: %v", err)
			continue
		}

		if len(candles) > 0 {
			candleChan <- candles[0]
		}
	}
}

func collectSignals(client *exchange.Client, signalChan chan analyzer.Signal) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	timeframes := []string{"5m", "30m", "1h"}

	for range ticker.C {
		for _, tf := range timeframes {
			candles, err := client.GetCandles("BTCUSDT", tf, 100)
			if err != nil {
				log.Printf("Error fetching %s candles: %v", tf, err)
				continue
			}

			// Analyze patterns and generate signals
			patterns := analyzer.AnalyzePatterns(candles)
			prediction := analyzer.PredictPrice(candles, tf)
			risk := calculateRisk(candles)

			// Create and send signal
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
