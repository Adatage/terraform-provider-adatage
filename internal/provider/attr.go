package provider

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The provider describes every resource and data source as a list of Attr and a few functions that call the API. The engine in
// resource.go turns that into a Terraform schema, plans and state. Values travel as Go values: string, int64, bool, []any,
// map[string]any, nil for null.

type Kind int

const (
	String Kind = iota
	Int
	Bool
	StringList
	StringMap
	// List is a list of objects described by Nested; Object is one such object.
	List
	Object
)

type Attr struct {
	Name string
	Desc string
	Kind Kind

	Required, Optional, Computed bool
	// ForceNew replaces the resource when the attribute changes.
	ForceNew bool
	// Stable computed values do not change by themselves, so a plan keeps them.
	Stable    bool
	Sensitive bool
	// Local attributes are not read back from the API: they hold what the configuration (or Create) said.
	Local bool
	// Path is where the API's JSON object has the value, dotted ("spec.sizeGib", "nics.0.status.ipv4Address"); empty means the name in
	// camelCase.
	Path string
	// Enum is the prefix of the API's enum values; Terraform sees them lower case without it (IP_TYPE_IPV4_ONLY is ipv4_only).
	Enum   string
	Nested []Attr
}

func camel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func (a Attr) path() string {
	if a.Path != "" {
		return a.Path
	}
	return camel(a.Name)
}

func (a Attr) tfType() tftypes.Type {
	switch a.Kind {
	case Int:
		return tftypes.Number
	case Bool:
		return tftypes.Bool
	case StringList:
		return tftypes.List{ElementType: tftypes.String}
	case StringMap:
		return tftypes.Map{ElementType: tftypes.String}
	case List:
		return tftypes.List{ElementType: objectType(a.Nested)}
	case Object:
		return objectType(a.Nested)
	}
	return tftypes.String
}

func objectType(attrs []Attr) tftypes.Object {
	types := map[string]tftypes.Type{}
	for _, a := range attrs {
		types[a.Name] = a.tfType()
	}
	return tftypes.Object{AttributeTypes: types}
}

type unknownValue struct{}

var unknown = unknownValue{}

func isUnknown(v any) bool { _, ok := v.(unknownValue); return ok }

// Values is an object: attribute name to Go value.
type Values map[string]any

func (v Values) known(name string) any {
	x := v[name]
	if isUnknown(x) {
		return nil
	}
	return x
}

func (v Values) Has(name string) bool { return v.known(name) != nil }

func (v Values) Str(name string) string {
	switch x := v.known(name).(type) {
	case string:
		return x
	case int64:
		return strconv.FormatInt(x, 10)
	}
	return ""
}

func (v Values) Int(name string) int64 {
	switch x := v.known(name).(type) {
	case int64:
		return x
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

func (v Values) Bool(name string) bool { b, _ := v.known(name).(bool); return b }

func (v Values) List(name string) []any { l, _ := v.known(name).([]any); return l }

func (v Values) Strings(name string) []string {
	var out []string
	for _, x := range v.List(name) {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (v Values) Map(name string) map[string]any { m, _ := v.known(name).(map[string]any); return m }

func (v Values) Objs(name string) []Values {
	var out []Values
	for _, x := range v.List(name) {
		if m, ok := x.(map[string]any); ok {
			out = append(out, Values(m))
		}
	}
	return out
}

func (v Values) Obj(name string) Values { return Values(v.Map(name)) }

// toGo converts a Terraform value. Unknown values become the unknown marker.
func toGo(v tftypes.Value) (any, error) {
	if !v.IsKnown() {
		return unknown, nil
	}
	if v.IsNull() {
		return nil, nil
	}
	t := v.Type()
	switch {
	case t.Is(tftypes.String):
		var s string
		err := v.As(&s)
		return s, err
	case t.Is(tftypes.Number):
		var f big.Float
		if err := v.As(&f); err != nil {
			return nil, err
		}
		n, _ := f.Int64()
		return n, nil
	case t.Is(tftypes.Bool):
		var b bool
		err := v.As(&b)
		return b, err
	case t.Is(tftypes.List{}), t.Is(tftypes.Set{}):
		var items []tftypes.Value
		if err := v.As(&items); err != nil {
			return nil, err
		}
		out := make([]any, 0, len(items))
		for _, it := range items {
			x, err := toGo(it)
			if err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		return out, nil
	case t.Is(tftypes.Map{}), t.Is(tftypes.Object{}):
		var items map[string]tftypes.Value
		if err := v.As(&items); err != nil {
			return nil, err
		}
		out := map[string]any{}
		for k, it := range items {
			x, err := toGo(it)
			if err != nil {
				return nil, err
			}
			out[k] = x
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported type %s", t)
}

// fromGo builds a Terraform value of type t.
func fromGo(t tftypes.Type, x any) (tftypes.Value, error) {
	if x == nil {
		return tftypes.NewValue(t, nil), nil
	}
	if isUnknown(x) {
		return tftypes.NewValue(t, tftypes.UnknownValue), nil
	}
	switch {
	case t.Is(tftypes.String):
		if s, ok := x.(string); ok {
			return tftypes.NewValue(t, s), nil
		}
		return tftypes.NewValue(t, fmt.Sprint(x)), nil
	case t.Is(tftypes.Number):
		switch n := x.(type) {
		case int64:
			return tftypes.NewValue(t, n), nil
		case int:
			return tftypes.NewValue(t, int64(n)), nil
		case float64:
			return tftypes.NewValue(t, big.NewFloat(n)), nil
		}
		return tftypes.Value{}, fmt.Errorf("%v is not a number", x)
	case t.Is(tftypes.Bool):
		b, _ := x.(bool)
		return tftypes.NewValue(t, b), nil
	case t.Is(tftypes.List{}):
		items, _ := x.([]any)
		out := make([]tftypes.Value, 0, len(items))
		for _, it := range items {
			v, err := fromGo(t.(tftypes.List).ElementType, it)
			if err != nil {
				return tftypes.Value{}, err
			}
			out = append(out, v)
		}
		return tftypes.NewValue(t, out), nil
	case t.Is(tftypes.Map{}):
		items, _ := x.(map[string]any)
		out := map[string]tftypes.Value{}
		for k, it := range items {
			v, err := fromGo(t.(tftypes.Map).ElementType, it)
			if err != nil {
				return tftypes.Value{}, err
			}
			out[k] = v
		}
		return tftypes.NewValue(t, out), nil
	case t.Is(tftypes.Object{}):
		items, _ := x.(map[string]any)
		out := map[string]tftypes.Value{}
		for k, at := range t.(tftypes.Object).AttributeTypes {
			v, err := fromGo(at, items[k])
			if err != nil {
				return tftypes.Value{}, err
			}
			out[k] = v
		}
		return tftypes.NewValue(t, out), nil
	}
	return tftypes.Value{}, fmt.Errorf("unsupported type %s", t)
}

func lookup(obj any, path string) (any, bool) {
	cur := obj
	for _, part := range strings.Split(path, ".") {
		switch c := cur.(type) {
		case map[string]any:
			next, ok := c[part]
			if !ok {
				return nil, false
			}
			cur = next
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(c) {
				return nil, false
			}
			cur = c[i]
		default:
			return nil, false
		}
	}
	return cur, cur != nil
}

func zero(a Attr) any {
	switch a.Kind {
	case Int:
		return int64(0)
	case Bool:
		return false
	case StringList, List:
		return []any{}
	case StringMap:
		return map[string]any{}
	case Object:
		return nil
	}
	return ""
}

func fromEnum(prefix, s string) string { return strings.ToLower(strings.TrimPrefix(s, prefix)) }

// Enum turns a Terraform value into the API's.
func Enum(prefix, s string) string {
	if s == "" {
		return ""
	}
	return prefix + strings.ToUpper(s)
}

func asInt(x any) (int64, bool) {
	switch n := x.(type) {
	case float64:
		return int64(n), true
	case string:
		v, err := strconv.ParseInt(n, 10, 64)
		return v, err == nil
	case int64:
		return n, true
	}
	return 0, false
}

// readObject turns the API's JSON object into the values of attrs. prior holds what the plan or state had.
func readObject(attrs []Attr, obj map[string]any, prior Values) Values {
	return readValues(attrs, obj, prior, false)
}

// readValues reads the attributes; inside a list or object everything the API returns is taken, as there is no prior value to match.
func readValues(attrs []Attr, obj map[string]any, prior Values, nested bool) Values {
	out := Values{}
	for _, a := range attrs {
		p := prior.known(a.Name)
		if a.Local {
			out[a.Name] = p
			continue
		}
		if !nested && p == nil && !a.Computed && !a.Required {
			out[a.Name] = nil
			continue
		}
		raw, ok := lookup(obj, a.path())
		if !ok {
			out[a.Name] = zero(a)
			continue
		}
		out[a.Name] = convert(a, raw)
	}
	return out
}

func convert(a Attr, raw any) any {
	switch a.Kind {
	case Int:
		if n, ok := asInt(raw); ok {
			return n
		}
		return int64(0)
	case Bool:
		b, _ := raw.(bool)
		return b
	case StringList:
		out := []any{}
		for _, x := range asList(raw) {
			if s, ok := x.(string); ok {
				out = append(out, enumOrPlain(a, s))
			}
		}
		return out
	case StringMap:
		out := map[string]any{}
		if m, ok := raw.(map[string]any); ok {
			for k, v := range m {
				out[k], _ = v.(string)
			}
		}
		return out
	case List:
		out := []any{}
		for _, x := range asList(raw) {
			if m, ok := x.(map[string]any); ok {
				out = append(out, map[string]any(readValues(a.Nested, m, nil, true)))
			}
		}
		return out
	case Object:
		if m, ok := raw.(map[string]any); ok {
			return map[string]any(readValues(a.Nested, m, nil, true))
		}
		return nil
	}
	s, _ := raw.(string)
	if s == "" {
		if n, ok := asInt(raw); ok {
			return strconv.FormatInt(n, 10)
		}
	}
	return enumOrPlain(a, s)
}

func asList(raw any) []any { l, _ := raw.([]any); return l }

func enumOrPlain(a Attr, s string) string {
	if a.Enum != "" {
		return fromEnum(a.Enum, s)
	}
	return s
}
