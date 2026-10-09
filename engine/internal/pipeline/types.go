// Package pipeline defines the core trading types, the stage interfaces a
// strategy pipeline is built from, typed parameter schemas, and the registry
// that maps a config `type` to a component constructor.
package pipeline

import (
	"fmt"
	"time"
)

// Timeframe is a bar period such as M15 or H1.
type Timeframe string

const (
	M1  Timeframe = "M1"
	M5  Timeframe = "M5"
	M15 Timeframe = "M15"
	M30 Timeframe = "M30"
	H1  Timeframe = "H1"
	H4  Timeframe = "H4"
	D1  Timeframe = "D1"
)

var timeframeDurations = map[Timeframe]time.Duration{
	M1:  time.Minute,
	M5:  5 * time.Minute,
	M15: 15 * time.Minute,
	M30: 30 * time.Minute,
	H1:  time.Hour,
	H4:  4 * time.Hour,
	D1:  24 * time.Hour,
}

// ParseTimeframe validates a timeframe name.
func ParseTimeframe(s string) (Timeframe, error) {
	tf := Timeframe(s)
	if _, ok := timeframeDurations[tf]; !ok {
		return "", fmt.Errorf("unknown timeframe %q (want M1, M5, M15, M30, H1, H4 or D1)", s)
	}
	return tf, nil
}

// Duration is the length of one bar.
func (tf Timeframe) Duration() time.Duration { return timeframeDurations[tf] }

// Bar is one OHLC candle. Time is the bar's open time in UTC.
type Bar struct {
	Symbol     string
	TF         Timeframe
	Time       time.Time
	Open       float64
	High       float64
	Low        float64
	Close      float64
	TickVolume int64
	Spread     int // in points, as recorded by MT5
}

// CloseTime is when the bar completes.
func (b Bar) CloseTime() time.Time { return b.Time.Add(b.TF.Duration()) }

// Side is a trade direction.
type Side int8

const (
	Flat  Side = 0
	Long  Side = 1
	Short Side = -1
)

func (s Side) String() string {
	switch s {
	case Long:
		return "long"
	case Short:
		return "short"
	default:
		return "flat"
	}
}

// Opposite returns the other direction; Flat stays Flat.
func (s Side) Opposite() Side { return -s }

// Instrument describes the symbol a pipeline trades.
type Instrument struct {
	Symbol       string
	Digits       int
	PipSize      float64 // 0.0001 for most pairs, 0.01 for JPY pairs and gold
	ContractSize float64 // units per 1.0 lot
}

// Signal is a stage-3 output: what the strategy wants, before sizing.
type Signal struct {
	Symbol   string
	Side     Side
	Strength float64 // 0..1
	Time     time.Time
	Reason   string
}

// Target is a sized trade with its exits, before execution.
type Target struct {
	Symbol     string
	Side       Side
	Lots       float64
	StopLoss   float64 // price, 0 = none
	TakeProfit float64 // price, 0 = none
	Reason     string
}

// OrderType is how an order is placed.
type OrderType string

const (
	Market OrderType = "market"
	Limit  OrderType = "limit"
	Stop   OrderType = "stop"
)

// Action is what an order intent does.
type Action string

const (
	Open   Action = "open"
	Close  Action = "close"
	Modify Action = "modify"
)

// OrderIntent is what a pipeline asks a broker to do.
type OrderIntent struct {
	ID         string
	RunID      string
	Action     Action
	Symbol     string
	Side       Side
	Lots       float64
	Type       OrderType
	Price      float64 // limit/stop price; 0 for market
	StopLoss   float64
	TakeProfit float64
	PositionID string // for Close and Modify
	Reason     string
}

// Fill is a broker's report that an order executed.
type Fill struct {
	OrderID    string
	PositionID string
	Symbol     string
	Side       Side
	Lots       float64
	Price      float64
	Time       time.Time
	Commission float64
}

// Position is an open trade.
type Position struct {
	ID         string
	Symbol     string
	Side       Side
	Lots       float64
	EntryPrice float64
	EntryTime  time.Time
	StopLoss   float64
	TakeProfit float64
}

// ExitReason records why a position closed.
type ExitReason string

const (
	ExitSignal     ExitReason = "signal"
	ExitStopLoss   ExitReason = "stop_loss"
	ExitTakeProfit ExitReason = "take_profit"
	ExitTime       ExitReason = "time"
	ExitManual     ExitReason = "manual"
	ExitEndOfData  ExitReason = "end_of_data"
)

// Trade is a closed position with its result in account currency.
type Trade struct {
	PositionID string
	Symbol     string
	Side       Side
	Lots       float64
	EntryTime  time.Time
	EntryPrice float64
	ExitTime   time.Time
	ExitPrice  float64
	Pips       float64
	Gross      float64
	Costs      float64
	Net        float64
	ExitReason ExitReason
}

// AccountState is a snapshot of a run's ledger.
type AccountState struct {
	Currency   string
	Balance    float64
	Equity     float64
	MarginUsed float64
}
