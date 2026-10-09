package pipeline

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// ParamType is the kind of value a parameter holds.
type ParamType string

const (
	TypeInt     ParamType = "int"
	TypeFloat   ParamType = "float"
	TypeString  ParamType = "string"
	TypeBool    ParamType = "bool"
	TypeEnum    ParamType = "enum"
	TypeFeature ParamType = "feature" // the name of a feature in the same strategy
)

// Field declares one parameter. Schemas are ordered so the UI can render
// fields in a stable order.
type Field struct {
	Name        string    `json:"name"`
	Type        ParamType `json:"type"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Default     any       `json:"default,omitempty"`
	Min         *float64  `json:"min,omitempty"`
	Max         *float64  `json:"max,omitempty"`
	Options     []string  `json:"options,omitempty"`
	Unit        string    `json:"unit,omitempty"`
	Required    bool      `json:"required,omitempty"`
}

// Schema is the ordered list of a component's parameters.
type Schema []Field

// Bound is a helper for Field.Min and Field.Max.
func Bound(v float64) *float64 { return &v }

// Params holds parameter values by name.
type Params map[string]any

// FieldError is one validation problem, located by path.
type FieldError struct {
	Path string
	Msg  string
}

func (e FieldError) Error() string { return e.Path + ": " + e.Msg }

// Errors collects every validation problem instead of stopping at the first.
type Errors []FieldError

func (es Errors) Error() string {
	msgs := make([]string, len(es))
	for i, e := range es {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

// Prefix returns the errors with path prepended to each path.
func (es Errors) Prefix(path string) Errors {
	out := make(Errors, len(es))
	for i, e := range es {
		out[i] = FieldError{Path: joinPath(path, e.Path), Msg: e.Msg}
	}
	return out
}

func joinPath(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "." + b
	}
}

// Field returns the named field.
func (s Schema) Field(name string) (Field, bool) {
	for _, f := range s {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

// Resolve checks raw values against the schema, fills defaults, and
// converts values to their canonical Go types (int, float64, string, bool).
// Unknown keys are errors so that typos never pass silently.
func (s Schema) Resolve(raw Params) (Params, error) {
	var errs Errors
	out := make(Params, len(s))
	for name := range raw {
		if _, ok := s.Field(name); !ok {
			errs = append(errs, FieldError{Path: name, Msg: "unknown parameter"})
		}
	}
	for _, f := range s {
		v, present := raw[f.Name]
		if !present || v == nil {
			if f.Default != nil {
				v = f.Default
			} else if f.Required {
				errs = append(errs, FieldError{Path: f.Name, Msg: "required"})
				continue
			} else {
				continue
			}
		}
		cv, err := f.coerce(v)
		if err != nil {
			errs = append(errs, FieldError{Path: f.Name, Msg: err.Error()})
			continue
		}
		out[f.Name] = cv
	}
	if len(errs) > 0 {
		slices.SortFunc(errs, func(a, b FieldError) int { return strings.Compare(a.Path, b.Path) })
		return nil, errs
	}
	return out, nil
}

func (f Field) coerce(v any) (any, error) {
	switch f.Type {
	case TypeInt:
		n, err := toFloat(v)
		if err != nil {
			return nil, err
		}
		if n != math.Trunc(n) {
			return nil, fmt.Errorf("must be a whole number, got %v", v)
		}
		if err := f.checkRange(n); err != nil {
			return nil, err
		}
		return int(n), nil
	case TypeFloat:
		n, err := toFloat(v)
		if err != nil {
			return nil, err
		}
		if err := f.checkRange(n); err != nil {
			return nil, err
		}
		return n, nil
	case TypeBool:
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("must be true or false, got %v", v)
		}
		return b, nil
	case TypeString, TypeFeature:
		str, ok := v.(string)
		if !ok || str == "" {
			return nil, fmt.Errorf("must be a non-empty string, got %v", v)
		}
		return str, nil
	case TypeEnum:
		str, ok := v.(string)
		if !ok || !slices.Contains(f.Options, str) {
			return nil, fmt.Errorf("must be one of %s, got %v", strings.Join(f.Options, ", "), v)
		}
		return str, nil
	default:
		return nil, fmt.Errorf("schema has unknown type %q", f.Type)
	}
}

func (f Field) checkRange(n float64) error {
	if f.Min != nil && n < *f.Min {
		return fmt.Errorf("must be at least %v, got %v", *f.Min, n)
	}
	if f.Max != nil && n > *f.Max {
		return fmt.Errorf("must be at most %v, got %v", *f.Max, n)
	}
	return nil
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case float32:
		return float64(n), nil
	case float64:
		return n, nil
	case string:
		// "${params.x}" substitution can leave numbers as strings.
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return 0, fmt.Errorf("must be a number, got %q", n)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("must be a number, got %v", v)
	}
}

// The getters below read values that Schema.Resolve has already checked and
// converted, so they do not return errors. A missing optional value returns
// the zero value.

func (p Params) Int(name string) int       { v, _ := p[name].(int); return v }
func (p Params) Float(name string) float64 { v, _ := p[name].(float64); return v }
func (p Params) String(name string) string { v, _ := p[name].(string); return v }
func (p Params) Bool(name string) bool     { v, _ := p[name].(bool); return v }
func (p Params) Has(name string) bool      { _, ok := p[name]; return ok }
