package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type AnyType struct{}

var AnyMetaType px.ObjectType

func init() {
	px.NewCollector = NewCollector
	px.NewTypedName = NewTypedName
	px.NewTypedName2 = newTypedName2
	px.TypedNameFromMapKey = typedNameFromMapKey

	AnyMetaType = newObjectType(`Pcore::AnyType`, `{}`, func(ctx px.Context, args []px.Value) px.Value {
		return DefaultAnyType()
	})
}

func DefaultAnyType() *AnyType {
	return anyTypeDefault
}

func (t *AnyType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *AnyType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*AnyType)
	return ok
}

func (t *AnyType) IsAssignable(o px.Type, g px.Guard) bool {
	return true
}

func (t *AnyType) IsInstance(v px.Value, g px.Guard) bool {
	return true
}

func (t *AnyType) MetaType() px.ObjectType {
	return AnyMetaType
}

func (t *AnyType) Name() string {
	return `Any`
}

func (t *AnyType) CanSerializeAsString() bool {
	return true
}

func (t *AnyType) SerializationString() string {
	return `Any`
}

func (t *AnyType) String() string {
	return `Any`
}

func (t *AnyType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *AnyType) PType() px.Type {
	return &TypeType{t}
}

var anyTypeDefault = &AnyType{}
