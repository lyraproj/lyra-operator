package types

import (
	"io"

	"github.com/lyraproj/pcore/px"
)

type (
	IteratorType struct {
		typ px.Type
	}
)

var iteratorTypeDefault = &IteratorType{typ: DefaultAnyType()}

var IteratorMetaType px.ObjectType

func init() {
	IteratorMetaType = newObjectType(`Pcore::IteratorType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx px.Context, args []px.Value) px.Value {
			return newIteratorType2(args...)
		})
}

func DefaultIteratorType() *IteratorType {
	return iteratorTypeDefault
}

func NewIteratorType(elementType px.Type) *IteratorType {
	if elementType == nil || elementType == anyTypeDefault {
		return DefaultIteratorType()
	}
	return &IteratorType{elementType}
}

func newIteratorType2(args ...px.Value) *IteratorType {
	switch len(args) {
	case 0:
		return DefaultIteratorType()
	case 1:
		containedType, ok := args[0].(px.Type)
		if !ok {
			panic(illegalArgumentType(`Iterator[]`, 0, `Type`, args[0]))
		}
		return NewIteratorType(containedType)
	default:
		panic(illegalArgumentCount(`Iterator[]`, `0 - 1`, len(args)))
	}
}

func (t *IteratorType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *IteratorType) Default() px.Type {
	return iteratorTypeDefault
}

func (t *IteratorType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*IteratorType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *IteratorType) Generic() px.Type {
	return NewIteratorType(px.GenericType(t.typ))
}

func (t *IteratorType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *IteratorType) IsAssignable(o px.Type, g px.Guard) bool {
	if it, ok := o.(*IteratorType); ok {
		return GuardedIsAssignable(t.typ, it.typ, g)
	}
	return false
}

func (t *IteratorType) IsInstance(o px.Value, g px.Guard) bool {
	if it, ok := o.(px.IteratorValue); ok {
		return GuardedIsInstance(t.typ, it.ElementType(), g)
	}
	return false
}

func (t *IteratorType) MetaType() px.ObjectType {
	return IteratorMetaType
}

func (t *IteratorType) Name() string {
	return `Iterator`
}

func (t *IteratorType) Parameters() []px.Value {
	if t.typ == DefaultAnyType() {
		return px.EmptyValues
	}
	return []px.Value{t.typ}
}

func (t *IteratorType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *IteratorType) SerializationString() string {
	return t.String()
}

func (t *IteratorType) String() string {
	return px.ToString2(t, None)
}

func (t *IteratorType) ElementType() px.Type {
	return t.typ
}

func (t *IteratorType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *IteratorType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IteratorType) PType() px.Type {
	return &TypeType{t}
}
