package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type VariantType struct {
	types []px.Type
}

var VariantMetaType px.ObjectType

func init() {
	VariantMetaType = newObjectType(`Pcore::VariantType`,
		`Pcore::AnyType {
	attributes => {
		types => Array[Type]
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newVariantType2(args...)
		})
}

func DefaultVariantType() *VariantType {
	return variantTypeDefault
}

func NewVariantType(types ...px.Type) px.Type {
	switch len(types) {
	case 0:
		return DefaultVariantType()
	case 1:
		return types[0]
	default:
		return &VariantType{types}
	}
}

func newVariantType2(args ...px.Value) px.Type {
	return newVariantType3(WrapValues(args))
}

func newVariantType3(args px.List) px.Type {
	var variants []px.Type
	var failIdx int

	switch args.Len() {
	case 0:
		return DefaultVariantType()
	case 1:
		first := args.At(0)
		switch first := first.(type) {
		case px.Type:
			return first
		case *Array:
			return newVariantType3(first)
		default:
			panic(illegalArgumentType(`Variant[]`, 0, `Type or Array[Type]`, args.At(0)))
		}
	default:
		variants, failIdx = toTypes(args)
		if failIdx >= 0 {
			panic(illegalArgumentType(`Variant[]`, failIdx, `Type`, args.At(failIdx)))
		}
	}
	return &VariantType{variants}
}

func (t *VariantType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	for _, c := range t.types {
		c.Accept(v, g)
	}
}

func (t *VariantType) Equals(o interface{}, g px.Guard) bool {
	ot, ok := o.(*VariantType)
	return ok && len(t.types) == len(ot.types) && px.IncludesAll(t.types, ot.types, g)
}

func (t *VariantType) Generic() px.Type {
	return &VariantType{UniqueTypes(alterTypes(t.types, generalize))}
}

func (t *VariantType) Default() px.Type {
	return variantTypeDefault
}

func (t *VariantType) IsAssignable(o px.Type, g px.Guard) bool {
	for _, v := range t.types {
		if GuardedIsAssignable(v, o, g) {
			return true
		}
	}
	return false
}

func (t *VariantType) IsInstance(o px.Value, g px.Guard) bool {
	for _, v := range t.types {
		if GuardedIsInstance(v, o, g) {
			return true
		}
	}
	return false
}

func (t *VariantType) MetaType() px.ObjectType {
	return VariantMetaType
}

func (t *VariantType) Name() string {
	return `Variant`
}

func (t *VariantType) Parameters() []px.Value {
	if len(t.types) == 0 {
		return px.EmptyValues
	}
	ps := make([]px.Value, len(t.types))
	for idx, t := range t.types {
		ps[idx] = t
	}
	return ps
}

func (t *VariantType) Resolve(c px.Context) px.Type {
	rts := make([]px.Type, len(t.types))
	for i, ts := range t.types {
		rts[i] = resolve(c, ts)
	}
	t.types = rts
	return t
}

func (t *VariantType) CanSerializeAsString() bool {
	for _, v := range t.types {
		if !canSerializeAsString(v) {
			return false
		}
	}
	return true
}

func (t *VariantType) SerializationString() string {
	return t.String()
}

func (t *VariantType) String() string {
	return px.ToString2(t, None)
}

func (t *VariantType) Types() []px.Type {
	return t.types
}

func (t *VariantType) allAssignableTo(o px.Type, g px.Guard) bool {
	return allAssignableTo(t.types, o, g)
}

func (t *VariantType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *VariantType) PType() px.Type {
	return &TypeType{t}
}

var variantTypeDefault = &VariantType{types: []px.Type{}}

func allAssignableTo(types []px.Type, o px.Type, g px.Guard) bool {
	for _, v := range types {
		if !GuardedIsAssignable(o, v, g) {
			return false
		}
	}
	return true
}
