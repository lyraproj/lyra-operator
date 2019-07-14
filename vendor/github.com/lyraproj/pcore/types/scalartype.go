package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type ScalarType struct{}

var ScalarMetaType px.ObjectType

func init() {
	ScalarMetaType = newObjectType(`Pcore::ScalarType`, `Pcore::AnyType{}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return DefaultScalarType()
		})
}

func DefaultScalarType() *ScalarType {
	return scalarTypeDefault
}

func (t *ScalarType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *ScalarType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*ScalarType)
	return ok
}

func (t *ScalarType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o.(type) {
	case *ScalarType, *ScalarDataType:
		return true
	default:
		return GuardedIsAssignable(stringTypeDefault, o, g) ||
			GuardedIsAssignable(numericTypeDefault, o, g) ||
			GuardedIsAssignable(booleanTypeDefault, o, g) ||
			GuardedIsAssignable(regexpTypeDefault, o, g)
	}
}

func (t *ScalarType) IsInstance(o px.Value, g px.Guard) bool {
	switch o.(type) {
	case stringValue, integerValue, floatValue, booleanValue, Timespan, *Timestamp, *SemVer, *Regexp:
		return true
	}
	return false
}

func (t *ScalarType) MetaType() px.ObjectType {
	return ScalarMetaType
}

func (t *ScalarType) Name() string {
	return `Scalar`
}

func (t *ScalarType) CanSerializeAsString() bool {
	return true
}

func (t *ScalarType) SerializationString() string {
	return t.String()
}

func (t *ScalarType) String() string {
	return `Scalar`
}

func (t *ScalarType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *ScalarType) PType() px.Type {
	return &TypeType{t}
}

var scalarTypeDefault = &ScalarType{}
