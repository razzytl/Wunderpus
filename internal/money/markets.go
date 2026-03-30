package money

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// MarketData represents market data for a trading pair.
type MarketData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume24h float64   `json:"volume_24h"`
	Change24h float64   `json:"change_24h"`
	Timestamp time.Time `json:"timestamp"`
}

// Order represents a trade order.
type Order struct {
	ID        string     `json:"id"`
	Symbol    string     `json:"symbol"`
	Type      string     `json:"type"` // "market", "limit"
	Side      string     `json:"side"` // "buy", "sell"
	Amount    float64    `json:"amount"`
	Price     float64    `json:"price"`
	Status    string     `json:"status"` // "pending", "filled", "canceled"
	CreatedAt time.Time  `json:"created_at"`
	FilledAt  *time.Time `json:"filled_at"`
}

// MarketIntelligence connects to exchanges and generates signals.
type MarketIntelligence struct {
	exchanges  []Exchange
	worldModel WorldModelQuery
	llm        LLMCaller
}

// Exchange interface for market data access.
type Exchange interface {
	GetMarketData(ctx context.Context, symbol string) (*MarketData, error)
	PlaceOrder(ctx context.Context, order *Order) error
	GetBalance(ctx context.Context) (map[string]float64, error)
}

// MarketConfig holds market engine configuration.
type MarketConfig struct {
	Enabled        bool
	PaperTrading   bool
	MaxPositionPct float64 // max % of portfolio per trade
	MaxDrawdownPct float64 // stop loss %
	Exchanges      []string
}

// NewMarketIntelligence creates a new market intelligence engine.
func NewMarketIntelligence(cfg MarketConfig, exchanges []Exchange, wm WorldModelQuery, llm LLMCaller) *MarketIntelligence {
	return &MarketIntelligence{
		exchanges:  exchanges,
		worldModel: wm,
		llm:        llm,
	}
}

// FetchPrices fetches current prices from connected exchanges.
func (m *MarketIntelligence) FetchPrices(ctx context.Context, symbols []string) (map[string]*MarketData, error) {
	prices := make(map[string]*MarketData)

	for _, exchange := range m.exchanges {
		for _, symbol := range symbols {
			data, err := exchange.GetMarketData(ctx, symbol)
			if err != nil {
				slog.Warn("market: failed to get data", "symbol", symbol, "error", err)
				continue
			}
			prices[symbol] = data
		}
	}

	return prices, nil
}

// GenerateSignal analyzes market data and generates a trading signal.
func (m *MarketIntelligence) GenerateSignal(ctx context.Context, symbol string, prices map[string]*MarketData) (*TradingSignal, error) {
	// Build context for signal generation
	prompt := "Analyze the following market data for " + symbol + " and generate a trading signal (buy/sell/hold) with reasoning:\n"
	for s, data := range prices {
		prompt += s + ": Price $" + formatFloat(data.Price) + ", 24h Change " + formatFloat(data.Change24h) + "%\n"
	}

	// Query world model for related news/events
	if m.worldModel != nil {
		if entities, err := m.worldModel.Context(ctx, "market events "+symbol); err == nil {
			prompt += "Related events: "
			for _, e := range entities {
				prompt += formatInterface(e) + ", "
			}
		}
	}

	req := LLMRequest{
		SystemPrompt: "You are a trading specialist. Analyze market data and generate signals.",
		UserPrompt:   prompt,
		Temperature:  0.3,
		MaxTokens:    500,
	}

	response, err := m.llm.Complete(req)
	if err != nil {
		return nil, err
	}

	// Parse response into signal (simplified)
	signal := &TradingSignal{
		Symbol:    symbol,
		Action:    "hold",
		Reasoning: response,
		CreatedAt: time.Now(),
	}

	return signal, nil
}

// TradingSignal represents a trading recommendation.
type TradingSignal struct {
	Symbol     string    `json:"symbol"`
	Action     string    `json:"action"` // "buy", "sell", "hold"
	Confidence float64   `json:"confidence"`
	Reasoning  string    `json:"reasoning"`
	CreatedAt  time.Time `json:"created_at"`
}

// PaperTrader simulates trades without real money.
type PaperTrader struct {
	balance     map[string]float64
	orders      []Order
	positions   map[string]float64
	maxDrawdown float64
	totalPnl    float64
}

// NewPaperTrader creates a new paper trader.
func NewPaperTrader(initialBalance float64) *PaperTrader {
	return &PaperTrader{
		balance:     map[string]float64{"USD": initialBalance},
		orders:      []Order{},
		positions:   make(map[string]float64),
		maxDrawdown: 0,
		totalPnl:    0,
	}
}

// ExecutePaperTrade simulates a trade and tracks P&L.
func (p *PaperTrader) ExecutePaperTrade(symbol string, side string, amount, price float64) error {
	order := Order{
		ID:        generateID(),
		Symbol:    symbol,
		Type:      "market",
		Side:      side,
		Amount:    amount,
		Price:     price,
		Status:    "filled",
		CreatedAt: time.Now(),
	}
	p.orders = append(p.orders, order)

	// Update balance
	cost := amount * price
	if side == "buy" {
		p.balance["USD"] -= cost
		p.positions[symbol] += amount
	} else {
		p.balance["USD"] += cost
		p.positions[symbol] -= amount
	}

	return nil
}

// CalculatePnL calculates current P&L.
func (p *PaperTrader) CalculatePnL(currentPrices map[string]float64) float64 {
	var totalValue float64
	for symbol, amount := range p.positions {
		if amount != 0 {
			price := currentPrices[symbol]
			totalValue += amount * price
		}
	}
	totalValue += p.balance["USD"]

	pnl := totalValue - 10000 // Assuming starting with $10k
	p.totalPnl = pnl
	return pnl
}

// CheckRiskLimits verifies trade is within risk limits.
func (p *PaperTrader) CheckRiskLimits(order *Order, currentPrices map[string]float64) bool {
	totalPortfolioValue := 0.0
	for symbol, amount := range p.positions {
		if price, ok := currentPrices[symbol]; ok {
			totalPortfolioValue += amount * price
		}
	}
	totalPortfolioValue += p.balance["USD"]

	// Check position size limit (max 2%)
	orderValue := order.Amount * order.Price
	if totalPortfolioValue > 0 && orderValue/totalPortfolioValue > 0.02 {
		slog.Warn("market: position size limit exceeded")
		return false
	}

	// Check drawdown limit (max 5%)
	if p.totalPnl < -500 { // 5% of $10k
		slog.Warn("market: drawdown limit hit, stopping trading")
		return false
	}

	return true
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func formatInterface(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
