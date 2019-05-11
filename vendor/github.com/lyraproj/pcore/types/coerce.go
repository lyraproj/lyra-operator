package types

import (
	"strings"

	"github.com/lyraproj/pcore/px"
)

// CoerceTo will deep coerce the given value into an instance of the given type t. The coercion will
// recurse down into hashes, arrays, and objects and take key, value, and attribute types into account.
//
// The label is used in potential type assertion errors. It should indicate what it is that is being
// coerced.
func CoerceTo(c px.Context, label string, typ px.Type, value px.Value) (result px.Value) {
	return coerceTo(c, []string{label}, typ, value)
}

type path []string

func (p path) with(n string) path {
	np := make(path, len(p)+1)
	copy(np, p)
	np[len(p)] = n
	return np
}

// CanCoerce responds true iv the given value can be coerced into an instance of the given type.
func CanCoerce(typ px.Type, value px.Value) bool {
	if typ.IsInstance(value, nil) {
		return true
	}

	if opt, ok := typ.(*OptionalType); ok {
		typ = opt.ContainedType()
	}

	switch t := typ.(type) {
	case *ArrayType:
		et := t.ElementType()
		if oa, ok := value.(*Array); ok {
			return oa.All(func(e px.Value) bool { return CanCoerce(et, e) })
		}
		return CanCoerce(et, value)
	case *HashType:
		kt := t.KeyType()
		vt := t.ValueType()
		if oh, ok := value.(*Hash); ok {
			return oh.All(func(v px.Value) bool {
				e := v.(px.MapEntry)
				if CanCoerce(kt, e.Key()) {
					return CanCoerce(vt, e.Value())
				}
				return false
			})
		}
		return false
	case *StructType:
		hm := t.HashedMembers()
		if oh, ok := value.(*Hash); ok {
			return oh.All(func(v px.Value) bool {
				e := v.(px.MapEntry)
				var s px.StringValue
				if s, ok = e.Key().(px.StringValue); ok {
					var se *StructElement
					if se, ok = hm[s.String()]; ok {
						return CanCoerce(se.Value(), e.Value())
					}
				}
				return false
			})
		}
		return false
	case px.ObjectType:
		ai := t.AttributesInfo()
		switch o := value.(type) {
		case *Array:
			for i, ca := range ai.Attributes() {
				if i >= o.Len() {
					return i <= ai.RequiredCount()
				}
				if !CanCoerce(ca.Type(), o.At(i)) {
					return false
				}
			}
		case *Hash:
			for _, ca := range ai.Attributes() {
				e, ok := o.GetEntry(ca.Name())
				if !ok {
					if !ca.HasValue() {
						return false
					}
				} else if !CanCoerce(ca.Type(), e.Value()) {
					return false
				}
			}
		default:
			return false
		}
		return true
	case *InitType:
		// Should have answered true to IsInstance above
		return false
	}
	return NewInitType(typ, emptyArray).IsInstance(value, nil)
}

func coerceTo(c px.Context, path path, typ px.Type, value px.Value) px.Value {
	if typ.IsInstance(value, nil) {
		return value
	}

	if opt, ok := typ.(*OptionalType); ok {
		typ = opt.ContainedType()
	}

	labelFunc := func() string { return strings.Join(path, `/`) }

	switch t := typ.(type) {
	case *ArrayType:
		et := t.ElementType()
		ep := path.with(`[]`)
		if oa, ok := value.(*Array); ok {
			value = oa.Map(func(e px.Value) px.Value { return coerceTo(c, ep, et, e) })
			if t.Size().IsInstance3(value.(*Array).Len()) {
				return value
			}
		}
		panic(px.MismatchError(labelFunc, t, value))
	case *HashType:
		kt := t.KeyType()
		vt := t.ValueType()
		kp := path.with(`key`)
		if oh, ok := value.(*Hash); ok {
			value = oh.MapEntries(func(e px.MapEntry) px.MapEntry {
				kv := coerceTo(c, kp, kt, e.Key())
				return WrapHashEntry(kv, coerceTo(c, path.with(kv.String()), vt, e.Value()))
			})
			if !t.Size().IsInstance3(value.(*Hash).Len()) {
				panic(px.MismatchError(labelFunc, t, value))
			}
			return value
		}
		panic(px.MismatchError(labelFunc, t, value))
	case *StructType:
		hm := t.HashedMembers()
		if oh, ok := value.(*Hash); ok {
			value = oh.MapEntries(func(e px.MapEntry) px.MapEntry {
				var s px.StringValue
				if s, ok = e.Key().(px.StringValue); ok {
					var se *StructElement
					if se, ok = hm[s.String()]; ok {
						return WrapHashEntry(s, coerceTo(c, path.with(s.String()), se.Value(), e.Value()))
					}
				}
				return e
			})
			return px.AssertInstance(labelFunc, t, value)
		}
		panic(px.MismatchError(labelFunc, t, value))
	case px.ObjectType:
		ai := t.AttributesInfo()
		if oh, ok := value.(*Hash); ok {
			el := make([]*HashEntry, 0, oh.Len())
			for _, ca := range ai.Attributes() {
				if e, ok := oh.GetEntry(ca.Name()); ok {
					el = append(el, WrapHashEntry(e.Key(), coerceTo(c, []string{ca.Label()}, ca.Type(), e.Value())))
				}
			}
			return newInstance(c, t, oh.Merge(WrapHash(el)))
		}
		panic(px.MismatchError(labelFunc, t, value))
	}
	return newInstance(c, typ, value)
}
