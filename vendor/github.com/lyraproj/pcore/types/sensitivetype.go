package types

import (
	"io"

	"github.com/lyraproj/pcore/utils"

	"github.com/lyraproj/pcore/px"
)

var sensitiveTypeDefault = &SensitiveType{typ: anyTypeDefault}

type (
	SensitiveType struct {
		typ px.Type
	}

	Sensitive struct {
		value px.Value
	}
)

var SensitiveMetaType px.ObjectType

func init() {
	SensitiveMetaType = newObjectType(`Pcore::SensitiveType`,
		`Pcore::AnyType {
	attributes => {
		type => {
			type => Optional[Type],
			value => Any
		},
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newSensitiveType2(args...)
		})

	newGoConstructor(`Sensitive`,
		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return WrapSensitive(args[0])
			})
		})
}

func DefaultSensitiveType() *SensitiveType {
	return sensitiveTypeDefault
}

func NewSensitiveType(containedType px.Type) *SensitiveType {
	if containedType == nil || containedType == anyTypeDefault {
		return DefaultSensitiveType()
	}
	return &SensitiveType{containedType}
}

func newSensitiveType2(args ...px.Value) *SensitiveType {
	switch len(args) {
	case 0:
		return DefaultSensitiveType()
	case 1:
		if containedType, ok := args[0].(px.Type); ok {
			return NewSensitiveType(containedType)
		}
		panic(illegalArgumentType(`Sensitive[]`, 0, `Type`, args[0]))
	default:
		panic(illegalArgumentCount(`Sensitive[]`, `0 or 1`, len(args)))
	}
}

func (t *SensitiveType) ContainedType() px.Type {
	return t.typ
}

func (t *SensitiveType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *SensitiveType) Default() px.Type {
	return DefaultSensitiveType()
}

func (t *SensitiveType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *SensitiveType) Generic() px.Type {
	return NewSensitiveType(px.GenericType(t.typ))
}

func (t *SensitiveType) IsAssignable(o px.Type, g px.Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *SensitiveType) IsInstance(o px.Value, g px.Guard) bool {
	if sv, ok := o.(*Sensitive); ok {
		return GuardedIsInstance(t.typ, sv.Unwrap(), g)
	}
	return false
}

func (t *SensitiveType) MetaType() px.ObjectType {
	return SensitiveMetaType
}

func (t *SensitiveType) Name() string {
	return `Sensitive`
}

func (t *SensitiveType) Parameters() []px.Value {
	if t.typ == DefaultAnyType() {
		return px.EmptyValues
	}
	return []px.Value{t.typ}
}

func (t *SensitiveType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *SensitiveType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *SensitiveType) SerializationString() string {
	return t.String()
}

func (t *SensitiveType) String() string {
	return px.ToString2(t, None)
}

func (t *SensitiveType) PType() px.Type {
	return &TypeType{t}
}

func (t *SensitiveType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func WrapSensitive(val px.Value) *Sensitive {
	return &Sensitive{val}
}

func (s *Sensitive) Equals(o interface{}, g px.Guard) bool {
	return false
}

func (s *Sensitive) String() string {
	return px.ToString2(s, None)
}

func (s *Sensitive) ToString(b io.Writer, f px.FormatContext, g px.RDetect) {
	utils.WriteString(b, `Sensitive [value redacted]`)
}

func (s *Sensitive) PType() px.Type {
	return NewSensitiveType(s.Unwrap().PType())
}

func (s *Sensitive) Unwrap() px.Value {
	return s.value
}
