package types

import (
	"io"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type TypeReferenceType struct {
	typeString string
}

var TypeReferenceMetaType px.ObjectType

func init() {
	TypeReferenceMetaType = newObjectType(`Pcore::TypeReference`,
		`Pcore::AnyType {
	attributes => {
		type_string => String[1]
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newTypeReferenceType2(args...)
		})
}

func DefaultTypeReferenceType() *TypeReferenceType {
	return typeReferenceTypeDefault
}

func NewTypeReferenceType(typeString string) *TypeReferenceType {
	return &TypeReferenceType{typeString}
}

func newTypeReferenceType2(args ...px.Value) *TypeReferenceType {
	switch len(args) {
	case 0:
		return DefaultTypeReferenceType()
	case 1:
		if str, ok := args[0].(stringValue); ok {
			return &TypeReferenceType{string(str)}
		}
		panic(illegalArgumentType(`TypeReference[]`, 0, `String`, args[0]))
	default:
		panic(illegalArgumentCount(`TypeReference[]`, `0 - 1`, len(args)))
	}
}

func (t *TypeReferenceType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *TypeReferenceType) Default() px.Type {
	return typeReferenceTypeDefault
}

func (t *TypeReferenceType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*TypeReferenceType); ok {
		return t.typeString == ot.typeString
	}
	return false
}

func (t *TypeReferenceType) Get(key string) (px.Value, bool) {
	switch key {
	case `type_string`:
		return stringValue(t.typeString), true
	default:
		return nil, false
	}
}

func (t *TypeReferenceType) IsAssignable(o px.Type, g px.Guard) bool {
	tr, ok := o.(*TypeReferenceType)
	return ok && t.typeString == tr.typeString
}

func (t *TypeReferenceType) IsInstance(o px.Value, g px.Guard) bool {
	return false
}

func (t *TypeReferenceType) MetaType() px.ObjectType {
	return TypeReferenceMetaType
}

func (t *TypeReferenceType) Name() string {
	return `TypeReference`
}

func (t *TypeReferenceType) CanSerializeAsString() bool {
	return true
}

func (t *TypeReferenceType) SerializationString() string {
	return t.String()
}

func (t *TypeReferenceType) String() string {
	return px.ToString2(t, None)
}

func (t *TypeReferenceType) Parameters() []px.Value {
	if *t == *typeReferenceTypeDefault {
		return px.EmptyValues
	}
	return []px.Value{stringValue(t.typeString)}
}

func (t *TypeReferenceType) Resolve(c px.Context) px.Type {
	r := c.ParseType(t.typeString)
	if rt, ok := r.(px.ResolvableType); ok {
		if tr, ok := rt.(*TypeReferenceType); ok && t.typeString == tr.typeString {
			panic(px.Error(px.UnresolvedType, issue.H{`typeString`: t.typeString}))
		}
		r = rt.Resolve(c)
	}
	return r
}

func (t *TypeReferenceType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TypeReferenceType) PType() px.Type {
	return &TypeType{t}
}

func (t *TypeReferenceType) TypeString() string {
	return t.typeString
}

var typeReferenceTypeDefault = &TypeReferenceType{`UnresolvedReference`}
