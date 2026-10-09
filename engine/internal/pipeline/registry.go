package pipeline

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Kind is the pipeline stage a component plugs into.
type Kind string

const (
	KindFeature   Kind = "feature"
	KindSignal    Kind = "signal"
	KindSizing    Kind = "sizing"
	KindStop      Kind = "stop_loss"
	KindTarget    Kind = "take_profit"
	KindExecution Kind = "execution"
)

// Factory describes one component type: its params and how to build it.
type Factory[T any] struct {
	Type        string
	Title       string
	Description string
	Schema      Schema
	// Needs lists what the component requires from the rest of the
	// pipeline, e.g. "stop_distance" for risk-based sizing.
	Needs []string
	New   func(p Params) (T, error)
}

// Build resolves raw params against the schema and constructs the component.
func (f Factory[T]) Build(raw Params) (T, Params, error) {
	var zero T
	p, err := f.Schema.Resolve(raw)
	if err != nil {
		return zero, nil, err
	}
	c, err := f.New(p)
	if err != nil {
		return zero, nil, err
	}
	return c, p, nil
}

// ComponentInfo is a registry entry as served to the UI and CLI.
type ComponentInfo struct {
	Kind        Kind     `json:"kind"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Needs       []string `json:"needs,omitempty"`
	Schema      Schema   `json:"params"`
}

type table[T any] struct {
	kind Kind
	m    map[string]Factory[T]
}

func (t *table[T]) add(f Factory[T]) {
	if f.Type == "" || f.New == nil {
		panic(fmt.Sprintf("pipeline: %s component needs a Type and New", t.kind))
	}
	if _, dup := t.m[f.Type]; dup {
		panic(fmt.Sprintf("pipeline: %s component %q registered twice", t.kind, f.Type))
	}
	seen := map[string]bool{}
	for _, fld := range f.Schema {
		if seen[fld.Name] {
			panic(fmt.Sprintf("pipeline: %s/%s declares param %q twice", t.kind, f.Type, fld.Name))
		}
		seen[fld.Name] = true
		if fld.Default != nil {
			if _, err := fld.coerce(fld.Default); err != nil {
				panic(fmt.Sprintf("pipeline: %s/%s param %q has an invalid default: %v", t.kind, f.Type, fld.Name, err))
			}
		}
	}
	t.m[f.Type] = f
}

func (t *table[T]) get(typ string) (Factory[T], error) {
	f, ok := t.m[typ]
	if !ok {
		names := make([]string, 0, len(t.m))
		for n := range t.m {
			names = append(names, n)
		}
		slices.Sort(names)
		return Factory[T]{}, fmt.Errorf("unknown %s type %q (registered: %s)", t.kind, typ, strings.Join(names, ", "))
	}
	return f, nil
}

func (t *table[T]) infos() []ComponentInfo {
	out := make([]ComponentInfo, 0, len(t.m))
	for _, f := range t.m {
		out = append(out, ComponentInfo{Kind: t.kind, Type: f.Type, Title: f.Title, Description: f.Description, Needs: f.Needs, Schema: f.Schema})
	}
	return out
}

// Registry holds every component type the engine knows. Config can only
// build what is registered here; there are no import paths in config.
type Registry struct {
	mu         sync.RWMutex
	features   table[Feature]
	signals    table[SignalGen]
	sizers     table[Sizer]
	stops      table[StopRule]
	targets    table[TargetRule]
	executions table[Executor]
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		features:   table[Feature]{kind: KindFeature, m: map[string]Factory[Feature]{}},
		signals:    table[SignalGen]{kind: KindSignal, m: map[string]Factory[SignalGen]{}},
		sizers:     table[Sizer]{kind: KindSizing, m: map[string]Factory[Sizer]{}},
		stops:      table[StopRule]{kind: KindStop, m: map[string]Factory[StopRule]{}},
		targets:    table[TargetRule]{kind: KindTarget, m: map[string]Factory[TargetRule]{}},
		executions: table[Executor]{kind: KindExecution, m: map[string]Factory[Executor]{}},
	}
}

// Default is the registry built-in components add themselves to in init().
var Default = NewRegistry()

func (r *Registry) AddFeature(f Factory[Feature]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.features.add(f)
}
func (r *Registry) AddSignal(f Factory[SignalGen]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.signals.add(f)
}
func (r *Registry) AddSizer(f Factory[Sizer])   { r.mu.Lock(); defer r.mu.Unlock(); r.sizers.add(f) }
func (r *Registry) AddStop(f Factory[StopRule]) { r.mu.Lock(); defer r.mu.Unlock(); r.stops.add(f) }
func (r *Registry) AddTarget(f Factory[TargetRule]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.targets.add(f)
}
func (r *Registry) AddExecution(f Factory[Executor]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executions.add(f)
}

func (r *Registry) Feature(typ string) (Factory[Feature], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.features.get(typ)
}

func (r *Registry) Signal(typ string) (Factory[SignalGen], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.signals.get(typ)
}

func (r *Registry) Sizer(typ string) (Factory[Sizer], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sizers.get(typ)
}

func (r *Registry) Stop(typ string) (Factory[StopRule], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stops.get(typ)
}

func (r *Registry) Target(typ string) (Factory[TargetRule], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.targets.get(typ)
}

func (r *Registry) Execution(typ string) (Factory[Executor], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.executions.get(typ)
}

// Components lists every registered component, sorted by kind then type.
func (r *Registry) Components() []ComponentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []ComponentInfo
	out = append(out, r.features.infos()...)
	out = append(out, r.signals.infos()...)
	out = append(out, r.sizers.infos()...)
	out = append(out, r.stops.infos()...)
	out = append(out, r.targets.infos()...)
	out = append(out, r.executions.infos()...)
	slices.SortFunc(out, func(a, b ComponentInfo) int {
		if c := strings.Compare(string(a.Kind), string(b.Kind)); c != 0 {
			return c
		}
		return strings.Compare(a.Type, b.Type)
	})
	return out
}
