package types

import (
	"io"

	"reflect"

	"github.com/lyraproj/pcore/px"
)

type NotUndefType struct {
	typ px.Type
}

var NotUndefMetaType px.ObjectType

func init() {
	NotUndefMetaType = newObjectType(`Pcore::NotUndefType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx px.Context, args []px.Value) px.Value {
			return newNotUndefType2(args...)
		})
}

func DefaultNotUndefType() *NotUndefType {
	return notUndefTypeDefault
}

func NewNotUndefType(containedType px.Type) *NotUndefType {
	if containedType == nil || containedType == anyTypeDefault {
		return DefaultNotUndefType()
	}
	return &NotUndefType{containedType}
}

func newNotUndefType2(args ...px.Value) *NotUndefType {
	switch len(args) {
	case 0:
		return DefaultNotUndefType()
	case 1:
		if containedType, ok := args[0].(px.Type); ok {
			return NewNotUndefType(containedType)
		}
		if containedType, ok := args[0].(stringValue); ok {
			return newNotUndefType3(string(containedType))
		}
		panic(illegalArgumentType(`NotUndef[]`, 0, `Variant[Type,String]`, args[0]))
	default:
		panic(illegalArgumentCount(`NotUndef[]`, `0 - 1`, len(args)))
	}
}

func newNotUndefType3(str string) *NotUndefType {
	return &NotUndefType{NewStringType(nil, str)}
}

func (t *NotUndefType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *NotUndefType) ContainedType() px.Type {
	return t.typ
}

func (t *NotUndefType) Default() px.Type {
	return notUndefTypeDefault
}

func (t *NotUndefType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*NotUndefType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *NotUndefType) Generic() px.Type {
	return NewNotUndefType(px.GenericType(t.typ))
}

func (t *NotUndefType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *NotUndefType) IsAssignable(o px.Type, g px.Guard) bool {
	return !GuardedIsAssignable(o, undefTypeDefault, g) && GuardedIsAssignable(t.typ, o, g)
}

func (t *NotUndefType) IsInstance(o px.Value, g px.Guard) bool {
	return o != undef && GuardedIsInstance(t.typ, o, g)
}

func (t *NotUndefType) MetaType() px.ObjectType {
	return NotUndefMetaType
}

func (t *NotUndefType) Name() string {
	return `NotUndef`
}

func (t *NotUndefType) Parameters() []px.Value {
	if t.typ == DefaultAnyType() {
		return px.EmptyValues
	}
	if str, ok := t.typ.(*vcStringType); ok && str.value != `` {
		return []px.Value{stringValue(str.value)}
	}
	return []px.Value{t.typ}
}

func (t *NotUndefType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *NotUndefType) ReflectType(c px.Context) (reflect.Type, bool) {
	return ReflectType(c, t.typ)
}

func (t *NotUndefType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *NotUndefType) SerializationString() string {
	return t.String()
}

func (t *NotUndefType) String() string {
	return px.ToString2(t, None)
}

func (t *NotUndefType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *NotUndefType) PType() px.Type {
	return &TypeType{t}
}

var notUndefTypeDefault = &NotUndefType{typ: anyTypeDefault}
