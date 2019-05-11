package types

import (
	"io"

	"github.com/lyraproj/pcore/utils"

	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type (
	UndefType struct{}

	// UndefValue is an empty struct because both type and value are known
	UndefValue struct{}
)

var undefTypeDefault = &UndefType{}

var UndefMetaType px.ObjectType

func init() {
	UndefMetaType = newObjectType(`Pcore::UndefType`, `Pcore::AnyType{}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return DefaultUndefType()
		})
}

func DefaultUndefType() *UndefType {
	return undefTypeDefault
}

func (t *UndefType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *UndefType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*UndefType)
	return ok
}

func (t *UndefType) IsAssignable(o px.Type, g px.Guard) bool {
	_, ok := o.(*UndefType)
	return ok
}

func (t *UndefType) IsInstance(o px.Value, g px.Guard) bool {
	return o == undef
}

func (t *UndefType) MetaType() px.ObjectType {
	return UndefMetaType
}

func (t *UndefType) Name() string {
	return `Undef`
}

func (t *UndefType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.Value{}.Type(), true
}

func (t *UndefType) CanSerializeAsString() bool {
	return true
}

func (t *UndefType) SerializationString() string {
	return t.String()
}

func (t *UndefType) String() string {
	return `Undef`
}

func (t *UndefType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UndefType) PType() px.Type {
	return &TypeType{t}
}

func WrapUndef() *UndefValue {
	return &UndefValue{}
}

func (uv *UndefValue) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*UndefValue)
	return ok
}

func (uv *UndefValue) Reflect(c px.Context) reflect.Value {
	return reflect.Value{}
}

func (uv *UndefValue) ReflectTo(c px.Context, value reflect.Value) {
	if !value.CanSet() {
		panic(px.Error(px.AttemptToSetUnsettable, issue.H{`kind`: value.Kind().String()}))
	}
	value.Set(reflect.Zero(value.Type()))
}

func (uv *UndefValue) String() string {
	return `undef`
}

func (uv *UndefValue) ToKey() px.HashKey {
	return px.HashKey([]byte{1, HkUndef})
}

func (uv *UndefValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	utils.WriteString(b, `undef`)
}

func (uv *UndefValue) PType() px.Type {
	return DefaultUndefType()
}
