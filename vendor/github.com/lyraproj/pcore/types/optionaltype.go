package types

import (
	"io"

	"reflect"

	"github.com/lyraproj/pcore/px"
)

type OptionalType struct {
	typ px.Type
}

var OptionalMetaType px.ObjectType

func init() {
	OptionalMetaType = newObjectType(`Pcore::OptionalType`,
		`Pcore::AnyType {
	attributes => {
		type => {
			type => Optional[Type],
			value => Any
		},
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newOptionalType2(args...)
		})
}

func DefaultOptionalType() *OptionalType {
	return optionalTypeDefault
}

func NewOptionalType(containedType px.Type) *OptionalType {
	if containedType == nil || containedType == anyTypeDefault {
		return DefaultOptionalType()
	}
	return &OptionalType{containedType}
}

func newOptionalType2(args ...px.Value) *OptionalType {
	switch len(args) {
	case 0:
		return DefaultOptionalType()
	case 1:
		if containedType, ok := args[0].(px.Type); ok {
			return NewOptionalType(containedType)
		}
		if containedType, ok := args[0].(stringValue); ok {
			return newOptionalType3(string(containedType))
		}
		panic(illegalArgumentType(`Optional[]`, 0, `Variant[Type,String]`, args[0]))
	default:
		panic(illegalArgumentCount(`Optional[]`, `0 - 1`, len(args)))
	}
}

func newOptionalType3(str string) *OptionalType {
	return &OptionalType{NewStringType(nil, str)}
}

func (t *OptionalType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *OptionalType) ContainedType() px.Type {
	return t.typ
}

func (t *OptionalType) Default() px.Type {
	return optionalTypeDefault
}

func (t *OptionalType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*OptionalType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *OptionalType) Generic() px.Type {
	return NewOptionalType(px.GenericType(t.typ))
}

func (t *OptionalType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *OptionalType) IsAssignable(o px.Type, g px.Guard) bool {
	return GuardedIsAssignable(o, undefTypeDefault, g) || GuardedIsAssignable(t.typ, o, g)
}

func (t *OptionalType) IsInstance(o px.Value, g px.Guard) bool {
	return o == undef || GuardedIsInstance(t.typ, o, g)
}

func (t *OptionalType) MetaType() px.ObjectType {
	return OptionalMetaType
}

func (t *OptionalType) Name() string {
	return `Optional`
}

func (t *OptionalType) Parameters() []px.Value {
	if t.typ == DefaultAnyType() {
		return px.EmptyValues
	}
	if str, ok := t.typ.(*vcStringType); ok && str.value != `` {
		return []px.Value{stringValue(str.value)}
	}
	return []px.Value{t.typ}
}

func (t *OptionalType) ReflectType(c px.Context) (reflect.Type, bool) {
	return ReflectType(c, t.typ)
}

func (t *OptionalType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *OptionalType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *OptionalType) SerializationString() string {
	return t.String()
}

func (t *OptionalType) String() string {
	return px.ToString2(t, None)
}

func (t *OptionalType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *OptionalType) PType() px.Type {
	return &TypeType{t}
}

var optionalTypeDefault = &OptionalType{typ: anyTypeDefault}
