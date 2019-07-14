package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type IterableType struct {
	typ px.Type
}

var IterableMetaType px.ObjectType

func init() {
	IterableMetaType = newObjectType(`Pcore::IterableType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx px.Context, args []px.Value) px.Value {
			return newIterableType2(args...)
		})
}

func DefaultIterableType() *IterableType {
	return iterableTypeDefault
}

func NewIterableType(elementType px.Type) *IterableType {
	if elementType == nil || elementType == anyTypeDefault {
		return DefaultIterableType()
	}
	return &IterableType{elementType}
}

func newIterableType2(args ...px.Value) *IterableType {
	switch len(args) {
	case 0:
		return DefaultIterableType()
	case 1:
		containedType, ok := args[0].(px.Type)
		if !ok {
			panic(illegalArgumentType(`Iterable[]`, 0, `Type`, args[0]))
		}
		return NewIterableType(containedType)
	default:
		panic(illegalArgumentCount(`Iterable[]`, `0 - 1`, len(args)))
	}
}

func (t *IterableType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *IterableType) Default() px.Type {
	return iterableTypeDefault
}

func (t *IterableType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*IterableType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *IterableType) Generic() px.Type {
	return NewIterableType(px.GenericType(t.typ))
}

func (t *IterableType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *IterableType) IsAssignable(o px.Type, g px.Guard) bool {
	var et px.Type
	switch o := o.(type) {
	case *ArrayType:
		et = o.ElementType()
	case *BinaryType:
		et = NewIntegerType(0, 255)
	case *HashType:
		et = o.EntryType()
	case *stringType, *vcStringType, *scStringType:
		et = OneCharStringType
	case *TupleType:
		return allAssignableTo(o.types, t.typ, g)
	default:
		return false
	}
	return GuardedIsAssignable(t.typ, et, g)
}

func (t *IterableType) IsInstance(o px.Value, g px.Guard) bool {
	if iv, ok := o.(px.Indexed); ok {
		return GuardedIsAssignable(t.typ, iv.ElementType(), g)
	}
	return false
}

func (t *IterableType) MetaType() px.ObjectType {
	return IterableMetaType
}

func (t *IterableType) Name() string {
	return `Iterable`
}

func (t *IterableType) Parameters() []px.Value {
	if t.typ == DefaultAnyType() {
		return px.EmptyValues
	}
	return []px.Value{t.typ}
}

func (t *IterableType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *IterableType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *IterableType) SerializationString() string {
	return t.String()
}

func (t *IterableType) String() string {
	return px.ToString2(t, None)
}

func (t *IterableType) ElementType() px.Type {
	return t.typ
}

func (t *IterableType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IterableType) PType() px.Type {
	return &TypeType{t}
}

var iterableTypeDefault = &IterableType{typ: DefaultAnyType()}
