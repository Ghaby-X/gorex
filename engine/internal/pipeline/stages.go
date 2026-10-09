package pipeline

import "time"

// Ctx is what every component sees on each step. One Ctx belongs to one
// run and one symbol; components never share it across goroutines.
type Ctx interface {
	Now() time.Time
	Instrument() Instrument
	// Bar is the latest closed bar.
	Bar() Bar
	// Feature returns a named feature's output series, as declared in the
	// strategy's features block.
	Feature(name string) (Series, bool)
	// Position returns the open position for this symbol, if any.
	Position() (Position, bool)
	Account() AccountState
}

// Feature turns bars into one numeric series (an indicator).
type Feature interface {
	// Update consumes a closed bar and returns the new value. ok is false
	// while the feature is still warming up.
	Update(b Bar) (v float64, ok bool)
	// Warmup is the number of bars needed before values are valid.
	Warmup() int
}

// SignalGen decides direction on each closed bar.
type SignalGen interface {
	OnBar(ctx Ctx) (Signal, bool)
}

// Sizer turns a signal into a volume in lots. stopDistance is the distance
// from entry to stop in price units, or 0 when the strategy has no stop.
type Sizer interface {
	Size(ctx Ctx, s Signal, stopDistance float64) (lots float64, err error)
}

// StopRule places the stop loss for a new position.
type StopRule interface {
	StopPrice(ctx Ctx, side Side, entry float64) float64
}

// TargetRule places the take profit for a new position. stop is the stop
// price (0 if none), so risk-reward targets can be measured from it.
type TargetRule interface {
	TargetPrice(ctx Ctx, side Side, entry, stop float64) float64
}

// Executor turns a sized target into order intents.
type Executor interface {
	Orders(ctx Ctx, t Target) []OrderIntent
}
