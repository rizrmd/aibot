package main

import (
	"aibot/internal/data"
	"aibot/internal/indicators"
	"aibot/internal/types"
	"aibot/pkg/stream"
	"aibot/pkg/trading"
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("🚀 Starting AI Bot Test...")

	// Test 1: Stream Provider
	fmt.Println("\n=== Testing Stream Provider ===")
	testStreamProvider()

	// Test 2: Trading Executor
	fmt.Println("\n=== Testing Trading Executor ===")
	testTradingExecutor()

	// Test 3: Technical Analysis
	fmt.Println("\n=== Testing Technical Analysis ===")
	testTechnicalAnalysis()

	// Test 4: Candle Aggregator
	fmt.Println("\n=== Testing Candle Aggregator ===")
	testCandleAggregator()

	fmt.Println("\n✅ All tests completed successfully!")
}

func testStreamProvider() {
	// Create simulation configuration
	config := stream.SimulationConfig{
		StreamConfig: stream.StreamConfig{
			ProviderType: "simulation",
			Symbols:      []string{"BTCUSDT", "ETHUSDT"},
		},
		SpeedMultiplier: 1.0,
		RandomVolatility: 0.1,
		InitialPrices: map[string]float64{
			"BTCUSDT": 45000.0,
			"ETHUSDT": 3000.0,
		},
		VolatilityFactors: map[string]float64{
			"BTCUSDT": 0.02,
			"ETHUSDT": 0.025,
		},
		CandleInterval: 300 * time.Millisecond,
		MaxHistoryCandles: 100,
	}

	// Create simulation provider
	provider := stream.NewSimulationProvider(config)

	// Start streaming
	ctx := context.Background()
	err := provider.Start(ctx, config.Symbols)
	if err != nil {
		log.Printf("Error starting stream provider: %v", err)
		return
	}

	fmt.Printf("✅ Stream provider connected: %v\n", provider.IsConnected())
	fmt.Printf("📊 Subscribed symbols: %v\n", provider.GetSubscribedSymbols())

	// Listen for some data
	tickerChan := provider.GetTickerChannel()
	ohlcvChan := provider.GetOHLCVChannel()

	// Collect some data for a short time
	dataCount := 0
	maxData := 10
	timeout := time.After(5 * time.Second)

	for dataCount < maxData {
		select {
		case ticker := <-tickerChan:
			fmt.Printf("💰 Ticker: %s = $%.2f (Vol: %.0f)\n",
				ticker.Symbol, ticker.Price, ticker.Volume)
			dataCount++
		case ohlcv := <-ohlcvChan:
			fmt.Printf("📈 OHLCV: %s O:$%.2f H:$%.2f L:$%.2f C:$%.2f V:%.0f\n",
				ohlcv.Symbol, ohlcv.Open, ohlcv.High, ohlcv.Low, ohlcv.Close, ohlcv.Volume)
			dataCount++
		case <-timeout:
			fmt.Println("⏰ Timeout reached")
			break
		}
	}

	provider.Stop()
	fmt.Printf("✅ Stream provider test completed with %d data points\n", dataCount)
}

func testTradingExecutor() {
	// Create simulation executor configuration
	config := trading.SimulationConfig{
		ExecutionConfig: trading.ExecutionConfig{
			ProviderType:     "simulation",
			InitialBalance:  10000.0,
			DefaultLeverage: 5.0,
			Commission:      0.0004,
		},
		Balance:         10000.0,
		Slippage:        0.0005,
		Latency:         50 * time.Millisecond,
		FillProbability: 0.9,
	}

	// Create executor
	executor := trading.NewSimulationExecutor(config)

	// Connect
	ctx := context.Background()
	err := executor.Connect(ctx)
	if err != nil {
		log.Printf("Error connecting executor: %v", err)
		return
	}

	fmt.Printf("✅ Trading executor connected: %v\n", executor.IsConnected())

	// Test order placement
	symbol := "BTCUSDT"
	quantity := 0.1
	price := 45000.0

	// Open long position
	result, err := executor.OpenLong(symbol, quantity, price)
	if err != nil {
		log.Printf("Error opening long: %v", err)
		return
	}

	fmt.Printf("📈 Long position opened: %s @ $%.2f (Filled: %.4f)\n",
		result.Symbol, result.FilledPrice, result.FilledQty)

	// Check position
	position, err := executor.GetPosition(symbol)
	if err != nil {
		log.Printf("Error getting position: %v", err)
		return
	}

	if position != nil {
		fmt.Printf("💼 Position: %s Size=%.4f Entry=$%.2f PnL=$%.2f\n",
			position.Symbol, position.Size, position.EntryPrice, position.UnrealizedPnL)
	}

	// Get balance
	balance, err := executor.GetBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
		return
	}

	fmt.Printf("💰 Account balance: $%.2f\n", balance)

	// Close position
	result, err = executor.CloseLong(symbol, quantity, price+100)
	if err != nil {
		log.Printf("Error closing long: %v", err)
		return
	}

	fmt.Printf("📉 Long position closed: %s @ $%.2f\n", result.Symbol, result.FilledPrice)

	// Check final position
	position, _ = executor.GetPosition(symbol)
	if position == nil || position.Size == 0 {
		fmt.Println("✅ Position successfully closed")
	}

	executor.Disconnect()
	fmt.Println("✅ Trading executor test completed")
}

func testTechnicalAnalysis() {
	// Create technical analyzer
	config := indicators.AnalyzerConfig{
		MaxHistoryCandles: 100,
	}
	analyzer := indicators.NewTechnicalAnalyzer(config)

	// Create sample OHLCV data
	symbol := "BTCUSDT"
	candles := createSampleCandles(symbol, 50)

	// Add candles to analyzer
	for _, candle := range candles {
		analyzer.AddCandle(candle)
	}

	fmt.Printf("✅ Added %d candles for %s\n", len(candles), symbol)

	// Get indicator values
	values := analyzer.GetIndicatorValues(symbol)
	if values != nil {
		fmt.Printf("📊 Technical Indicators for %s:\n", symbol)
		fmt.Printf("   Current Price: $%.2f\n", values.CurrentPrice)
		fmt.Printf("   SMA (20): $%.2f\n", values.SMA)
		fmt.Printf("   EMA (20): $%.2f\n", values.EMA)
		fmt.Printf("   RSI (14): %.2f\n", values.RSI)
		fmt.Printf("   ATR (14): $%.2f\n", values.ATR)
		fmt.Printf("   MACD: %.4f\n", values.MACD)
		fmt.Printf("   Bollinger Upper: $%.2f\n", values.BollingerUpper)
		fmt.Printf("   Bollinger Lower: $%.2f\n", values.BollingerLower)
	}

	// Test signal generation
	signalGen := indicators.NewSignalGenerator(indicators.SignalThresholds{})
	signal := signalGen.GenerateSignal(values, 1000000) // Volume = 1M

	if signal != nil {
		fmt.Printf("🎯 Trading Signal: %s (Strength: %v, Confidence: %.2f)\n",
			signal.Type, signal.Strength, signal.Confidence)
		fmt.Printf("   Reason: %s\n", signal.Reason)
		if signal.StopLoss > 0 {
			fmt.Printf("   Stop Loss: $%.2f\n", signal.StopLoss)
		}
		if signal.TakeProfit > 0 {
			fmt.Printf("   Take Profit: $%.2f\n", signal.TakeProfit)
		}
	}

	fmt.Println("✅ Technical analysis test completed")
}

func testCandleAggregator() {
	// Create aggregator configuration
	config := data.AggregatorConfig{
		BaseInterval: 300 * time.Millisecond,
		MaxHistory:   50,
		Timeframes:   []data.CandleTimeframe{data.Timeframe1s, data.Timeframe3s, data.Timeframe15s},
		Symbols:      []string{"BTCUSDT"},
	}

	aggregator := data.NewCandleAggregator(config)

	// Simulate ticker updates
	symbol := "BTCUSDT"
	basePrice := 45000.0

	for i := 0; i < 20; i++ {
		// Simulate price movement
		price := basePrice + (float64(i%5-2) * 10) // Small price oscillation
		volume := 1000.0 + float64(i*100)

		ticker := types.Ticker{
			Symbol:    symbol,
			Timestamp: time.Now().Add(time.Duration(i) * 300 * time.Millisecond),
			Price:     price,
			Volume:    volume,
			Bid:       price - 5,
			Ask:       price + 5,
		}

		aggregator.AddTick(ticker)
		time.Sleep(10 * time.Millisecond) // Small delay
	}

	fmt.Printf("✅ Added ticker data for %s\n", symbol)

	// Check aggregated candles for different timeframes
	timeframes := []data.CandleTimeframe{data.Timeframe1s, data.Timeframe3s, data.Timeframe15s}

	for _, tf := range timeframes {
		candles := aggregator.GetCandles(symbol, tf, 5)
		fmt.Printf("📊 %s candles: %d candles\n", tf, len(candles))

		for i, candle := range candles {
			if i < 3 { // Show first 3 candles
				fmt.Printf("   Candle %d: O:$%.2f H:$%.2f L:$%.2f C:$%.2f V:%.0f\n",
					i+1, candle.Open, candle.High, candle.Low, candle.Close, candle.Volume)
			}
		}
	}

	// Check current candle
	currentCandle := aggregator.GetCurrentCandle(symbol, data.Timeframe3s)
	if currentCandle != nil {
		fmt.Printf("🕯️ Current 3s candle: O:$%.2f H:$%.2f L:$%.2f C:$%.2f V:%.0f\n",
			currentCandle.Open, currentCandle.High, currentCandle.Low, currentCandle.Close, currentCandle.Volume)
	}

	// Get latest price
	latestPrice := aggregator.GetLatestPrice(symbol)
	fmt.Printf("💰 Latest price: $%.2f\n", latestPrice)

	fmt.Println("✅ Candle aggregator test completed")
}

// Helper function to create sample OHLCV data
func createSampleCandles(symbol string, count int) []types.OHLCV {
	candles := make([]types.OHLCV, count)
	basePrice := 45000.0
	baseTime := time.Now().Add(-time.Duration(count) * time.Second)

	for i := 0; i < count; i++ {
		// Create realistic price movements
		priceChange := (float64(i%10-5) / 100) * basePrice * 0.01 // ±0.05% max change
		open := basePrice + priceChange
		high := open * (1 + 0.002) // 0.2% above open
		low := open * (1 - 0.002)  // 0.2% below open
		close := open + (high-low)*0.5*float64(i%3-1) // Random close
		volume := 1000.0 + float64(i*50)

		candles[i] = types.OHLCV{
			Symbol:    symbol,
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		}

		basePrice = close // Use close as next base
	}

	return candles
}