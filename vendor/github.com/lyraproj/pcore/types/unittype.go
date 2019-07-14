package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type UnitType struct{}

var UnitMetaType px.ObjectType

func init() {
	UnitMetaType = newObjectType(`Pcore::UnitType`, `Pcore::AnyType{}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return DefaultUnitType()
		})

	newGoConstructor(`Unit`,
		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return args[0]
			})
		},
	)
}

func DefaultUnitType() *UnitType {
	return unitTypeDefault
}

func (t *UnitType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *UnitType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*UnitType)
	return ok
}

func (t *UnitType) IsAssignable(o px.Type, g px.Guard) bool {
	return true
}

func (t *UnitType) IsInstance(o px.Value, g px.Guard) bool {
	return true
}

func (t *UnitType) MetaType() px.ObjectType {
	return UnitMetaType
}

func (t *UnitType) Name() string {
	return `Unit`
}

func (t *UnitType) CanSerializeAsString() bool {
	return true
}

func (t *UnitType) SerializationString() string {
	return t.String()
}

func (t *UnitType) String() string {
	return `Unit`
}

func (t *UnitType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UnitType) PType() px.Type {
	return &TypeType{t}
}

var unitTypeDefault = &UnitType{}
