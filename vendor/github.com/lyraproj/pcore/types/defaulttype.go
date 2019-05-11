package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type (
	DefaultType struct{}

	// DefaultValue is an empty struct because both type and value are known
	DefaultValue struct{}
)

var defaultTypeDefault = &DefaultType{}

var DefaultMetaType px.ObjectType

func init() {
	DefaultMetaType = newObjectType(`Pcore::DefaultType`, `Pcore::AnyType{}`, func(ctx px.Context, args []px.Value) px.Value {
		return DefaultDefaultType()
	})
}

func DefaultDefaultType() *DefaultType {
	return defaultTypeDefault
}

func (t *DefaultType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *DefaultType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*DefaultType)
	return ok
}

func (t *DefaultType) IsAssignable(o px.Type, g px.Guard) bool {
	return o == defaultTypeDefault
}

func (t *DefaultType) IsInstance(o px.Value, g px.Guard) bool {
	_, ok := o.(*DefaultValue)
	return ok
}

func (t *DefaultType) MetaType() px.ObjectType {
	return DefaultMetaType
}

func (t *DefaultType) Name() string {
	return `Default`
}

func (t *DefaultType) CanSerializeAsString() bool {
	return true
}

func (t *DefaultType) SerializationString() string {
	return t.String()
}

func (t *DefaultType) String() string {
	return px.ToString2(t, None)
}

func (t *DefaultType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *DefaultType) PType() px.Type {
	return &TypeType{t}
}

func WrapDefault() *DefaultValue {
	return &DefaultValue{}
}

func (dv *DefaultValue) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*DefaultValue)
	return ok
}

func (dv *DefaultValue) ToKey() px.HashKey {
	return px.HashKey([]byte{1, HkDefault})
}

func (dv *DefaultValue) String() string {
	return `default`
}

func (dv *DefaultValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), dv.PType())
	switch f.FormatChar() {
	case 'd', 's', 'p':
		f.ApplyStringFlags(b, `default`, false)
	case 'D':
		f.ApplyStringFlags(b, `Default`, false)
	default:
		panic(s.UnsupportedFormat(dv.PType(), `dDsp`, f))
	}
}

func (dv *DefaultValue) PType() px.Type {
	return DefaultDefaultType()
}
