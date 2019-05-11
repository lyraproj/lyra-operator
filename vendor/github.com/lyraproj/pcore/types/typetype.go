package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type TypeType struct {
	typ px.Type
}

var typeTypeDefault = &TypeType{typ: anyTypeDefault}

var TypeMetaType px.ObjectType

func init() {
	TypeMetaType = newObjectType(`Pcore::TypeType`,
		`Pcore::AnyType {
	attributes => {
		type => {
			type => Optional[Type],
			value => Any
		},
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newTypeType2(args...)
		})

	newGoConstructor(`Type`,
		func(d px.Dispatch) {
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return c.ParseTypeValue(args[0])
			})
		},
		func(d px.Dispatch) {
			d.Param2(TypeObjectInitHash)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return MakeObjectType(``, nil, args[0].(px.OrderedMap), false).Resolve(c)
			})
		})
}

func DefaultTypeType() *TypeType {
	return typeTypeDefault
}

func NewTypeType(containedType px.Type) *TypeType {
	if containedType == nil || containedType == anyTypeDefault {
		return DefaultTypeType()
	}
	return &TypeType{containedType}
}

func newTypeType2(args ...px.Value) *TypeType {
	switch len(args) {
	case 0:
		return DefaultTypeType()
	case 1:
		if containedType, ok := args[0].(px.Type); ok {
			return NewTypeType(containedType)
		}
		panic(illegalArgumentType(`Type[]`, 0, `Type`, args[0]))
	default:
		panic(illegalArgumentCount(`Type[]`, `0 or 1`, len(args)))
	}
}

func (t *TypeType) ContainedType() px.Type {
	return t.typ
}

func (t *TypeType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *TypeType) Default() px.Type {
	return typeTypeDefault
}

func (t *TypeType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *TypeType) Generic() px.Type {
	return NewTypeType(px.GenericType(t.typ))
}

func (t *TypeType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *TypeType) IsAssignable(o px.Type, g px.Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *TypeType) IsInstance(o px.Value, g px.Guard) bool {
	if ot, ok := o.(px.Type); ok {
		return GuardedIsAssignable(t.typ, ot, g)
	}
	return false
}

func (t *TypeType) MetaType() px.ObjectType {
	return TypeMetaType
}

func (t *TypeType) Name() string {
	return `Type`
}

func (t *TypeType) Parameters() []px.Value {
	if t.typ == DefaultAnyType() {
		return px.EmptyValues
	}
	return []px.Value{t.typ}
}

func (t *TypeType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *TypeType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *TypeType) SerializationString() string {
	return t.String()
}

func (t *TypeType) String() string {
	return px.ToString2(t, None)
}

func (t *TypeType) PType() px.Type {
	return &TypeType{t}
}

func (t *TypeType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}
