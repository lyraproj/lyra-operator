package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type ScalarDataType struct{}

var ScalarDataMetaType px.ObjectType

func init() {
	ScalarDataMetaType = newObjectType(`Pcore::ScalarDataType`, `Pcore::ScalarType{}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return DefaultScalarDataType()
		})
}

func DefaultScalarDataType() *ScalarDataType {
	return scalarDataTypeDefault
}

func (t *ScalarDataType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *ScalarDataType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*ScalarDataType)
	return ok
}

func (t *ScalarDataType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o.(type) {
	case *ScalarDataType:
		return true
	default:
		return GuardedIsAssignable(stringTypeDefault, o, g) ||
			GuardedIsAssignable(integerTypeDefault, o, g) ||
			GuardedIsAssignable(booleanTypeDefault, o, g) ||
			GuardedIsAssignable(floatTypeDefault, o, g)
	}
}

func (t *ScalarDataType) IsInstance(o px.Value, g px.Guard) bool {
	switch o.(type) {
	case booleanValue, floatValue, integerValue, stringValue:
		return true
	}
	return false
}

func (t *ScalarDataType) MetaType() px.ObjectType {
	return ScalarDataMetaType
}

func (t *ScalarDataType) Name() string {
	return `ScalarData`
}

func (t *ScalarDataType) CanSerializeAsString() bool {
	return true
}

func (t *ScalarDataType) SerializationString() string {
	return t.String()
}

func (t *ScalarDataType) String() string {
	return `ScalarData`
}

func (t *ScalarDataType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *ScalarDataType) PType() px.Type {
	return &TypeType{t}
}

var scalarDataTypeDefault = &ScalarDataType{}
