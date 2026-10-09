package pipeline

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseTimeframe(t *testing.T) {
	tf, err := ParseTimeframe("M30")
	if err != nil || tf.Duration() != 30*time.Minute {
		t.Fatalf("M30: got %v, %v", tf.Duration(), err)
	}
	if _, err := ParseTimeframe("M2"); err == nil {
		t.Fatal("M2 should be rejected")
	}
	b := Bar{TF: H1, Time: time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)}
	if got := b.CloseTime(); !got.Equal(time.Date(2026, 1, 5, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("CloseTime = %v", got)
	}
}

func TestSide(t *testing.T) {
	if Long.Opposite() != Short || Short.Opposite() != Long || Flat.Opposite() != Flat {
		t.Fatal("Opposite is wrong")
	}
}

func TestRolling(t *testing.T) {
	r := NewRolling(3)
	for _, v := range []float64{1, 2, 3, 4} {
		r.Push(v)
	}
	if r.Len() != 3 || !r.Full() {
		t.Fatalf("Len = %d, Full = %v", r.Len(), r.Full())
	}
	want := []float64{4, 3, 2}
	for i, w := range want {
		if got := r.At(i); got != w {
			t.Errorf("At(%d) = %v, want %v", i, got, w)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("At past Len should panic")
		}
	}()
	r.At(3)
}

var testSchema = Schema{
	{Name: "period", Type: TypeInt, Default: 14, Min: Bound(2), Max: Bound(500)},
	{Name: "mult", Type: TypeFloat, Default: 1.5, Min: Bound(0.1)},
	{Name: "source", Type: TypeEnum, Default: "close", Options: []string{"open", "high", "low", "close"}},
	{Name: "series", Type: TypeFeature, Required: true},
	{Name: "enabled", Type: TypeBool},
}

func TestResolveDefaultsAndCoercion(t *testing.T) {
	p, err := testSchema.Resolve(Params{"period": 20.0, "mult": "2.5", "series": "fast_ma"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Int("period") != 20 || p.Float("mult") != 2.5 || p.String("source") != "close" || p.String("series") != "fast_ma" {
		t.Fatalf("resolved = %#v", p)
	}
	if p.Has("enabled") {
		t.Fatal("optional param without default should stay unset")
	}
}

func TestResolveErrors(t *testing.T) {
	_, err := testSchema.Resolve(Params{"period": 1, "mult": 2.5, "source": "median", "typo": 3})
	var errs Errors
	if !errors.As(err, &errs) {
		t.Fatalf("want Errors, got %v", err)
	}
	got := map[string]string{}
	for _, e := range errs {
		got[e.Path] = e.Msg
	}
	for path, frag := range map[string]string{
		"period": "at least 2",
		"source": "one of open, high, low, close",
		"series": "required",
		"typo":   "unknown parameter",
	} {
		if !strings.Contains(got[path], frag) {
			t.Errorf("%s: got %q, want it to contain %q", path, got[path], frag)
		}
	}
	if _, err := testSchema.Resolve(Params{"period": 14.5, "series": "x"}); err == nil || !strings.Contains(err.Error(), "whole number") {
		t.Errorf("fractional int: got %v", err)
	}
	if got := errs.Prefix("features.fast_ma").Error(); !strings.Contains(got, "features.fast_ma.period") {
		t.Errorf("Prefix: %s", got)
	}
}

type constFeature struct{}

func (constFeature) Update(Bar) (float64, bool) { return 1, true }
func (constFeature) Warmup() int                { return 0 }

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.AddFeature(Factory[Feature]{
		Type:   "const",
		Title:  "Constant",
		Schema: Schema{{Name: "period", Type: TypeInt, Default: 3, Min: Bound(1)}},
		New:    func(p Params) (Feature, error) { return constFeature{}, nil },
	})

	f, err := r.Feature("const")
	if err != nil {
		t.Fatal(err)
	}
	if _, p, err := f.Build(Params{}); err != nil || p.Int("period") != 3 {
		t.Fatalf("Build: %v %v", p, err)
	}
	if _, _, err := f.Build(Params{"period": 0}); err == nil {
		t.Fatal("Build should reject period 0")
	}
	if _, err := r.Feature("nope"); err == nil || !strings.Contains(err.Error(), "registered: const") {
		t.Fatalf("unknown type error should list registered types, got %v", err)
	}
	if infos := r.Components(); len(infos) != 1 || infos[0].Kind != KindFeature || infos[0].Type != "const" {
		t.Fatalf("Components = %+v", infos)
	}

	mustPanic(t, "duplicate type", func() {
		r.AddFeature(Factory[Feature]{Type: "const", New: func(Params) (Feature, error) { return constFeature{}, nil }})
	})
	mustPanic(t, "invalid default", func() {
		r.AddFeature(Factory[Feature]{
			Type:   "bad",
			Schema: Schema{{Name: "period", Type: TypeInt, Default: 0, Min: Bound(1)}},
			New:    func(Params) (Feature, error) { return constFeature{}, nil },
		})
	})
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	fn()
}
