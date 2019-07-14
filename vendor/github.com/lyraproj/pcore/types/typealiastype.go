package types

import (
	"fmt"
	"io"

	"github.com/lyraproj/pcore/utils"

	"github.com/lyraproj/pcore/px"
)

type TypeAliasType struct {
	name           string
	typeExpression *DeferredType
	resolvedType   px.Type
	loader         px.Loader
}

var TypeAliasMetaType px.ObjectType

func init() {
	TypeAliasMetaType = newObjectType(`Pcore::TypeAlias`,
		`Pcore::AnyType {
	attributes => {
		name => String[1],
		resolved_type => {
			type => Optional[Type],
			value => undef
		}
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newTypeAliasType2(args...)
		})
}

func DefaultTypeAliasType() *TypeAliasType {
	return typeAliasTypeDefault
}

// NewTypeAliasType creates a new TypeAliasType from a name and a typeExpression which
// must either be a *DeferredType, a parser.Expression, or nil. If it is nil, the
// resolved Type must be given.
func NewTypeAliasType(name string, typeExpression *DeferredType, resolvedType px.Type) *TypeAliasType {
	return &TypeAliasType{name, typeExpression, resolvedType, nil}
}

func newTypeAliasType2(args ...px.Value) *TypeAliasType {
	switch len(args) {
	case 0:
		return DefaultTypeAliasType()
	case 2:
		name, ok := args[0].(stringValue)
		if !ok {
			panic(illegalArgumentType(`TypeAlias`, 0, `String`, args[0]))
		}
		var pt px.Type
		if pt, ok = args[1].(px.Type); ok {
			return NewTypeAliasType(string(name), nil, pt)
		}
		if dt, ok := args[1].(*DeferredType); ok {
			return NewTypeAliasType(string(name), dt, nil)
		}
		panic(illegalArgumentType(`TypeAlias[]`, 1, `Type or Expression`, args[1]))
	default:
		panic(illegalArgumentCount(`TypeAlias[]`, `0 or 2`, len(args)))
	}
}

func (t *TypeAliasType) Accept(v px.Visitor, g px.Guard) {
	if g == nil {
		g = make(px.Guard)
	}
	if g.Seen(t, nil) {
		return
	}
	v(t)
	t.resolvedType.Accept(v, g)
}

func (t *TypeAliasType) Default() px.Type {
	return typeAliasTypeDefault
}

func (t *TypeAliasType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*TypeAliasType); ok && t.name == ot.name {
		if g == nil {
			g = make(px.Guard)
		}
		if g.Seen(t, ot) {
			return true
		}
		tr := t.resolvedType
		otr := ot.resolvedType
		return tr.Equals(otr, g)
	}
	return false
}

func (t *TypeAliasType) Get(key string) (px.Value, bool) {
	switch key {
	case `name`:
		return stringValue(t.name), true
	case `resolved_type`:
		return t.resolvedType, true
	default:
		return nil, false
	}
}

func (t *TypeAliasType) Loader() px.Loader {
	return t.loader
}

func (t *TypeAliasType) IsAssignable(o px.Type, g px.Guard) bool {
	if g == nil {
		g = make(px.Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsAssignable(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) IsInstance(o px.Value, g px.Guard) bool {
	if g == nil {
		g = make(px.Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsInstance(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) MetaType() px.ObjectType {
	return TypeAliasMetaType
}

func (t *TypeAliasType) Name() string {
	return t.name
}

func (t *TypeAliasType) Resolve(c px.Context) px.Type {
	if t.resolvedType == nil {
		t.resolvedType = t.typeExpression.Resolve(c)
		t.loader = c.Loader()
	}
	return t
}

func (t *TypeAliasType) ResolvedType() px.Type {
	if t.resolvedType == nil {
		panic(fmt.Sprintf("Reference to unresolved type '%s'", t.name))
	}
	return t.resolvedType
}

func (t *TypeAliasType) String() string {
	return px.ToString2(t, Expanded)
}

func (t *TypeAliasType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), t.PType())
	if t.name == `UnresolvedAlias` {
		utils.WriteString(b, `TypeAlias`)
	} else {
		utils.WriteString(b, t.name)
		if !(f.IsAlt() && f.FormatChar() == 'b') {
			return
		}
		if g == nil {
			g = make(px.RDetect)
		} else if g[t] {
			utils.WriteString(b, `<recursive reference>`)
			return
		}
		g[t] = true
		utils.WriteString(b, ` = `)

		// TODO: Need to be adjusted when included in TypeSet
		t.resolvedType.ToString(b, s, g)
		delete(g, t)
	}
}

func (t *TypeAliasType) PType() px.Type {
	return &TypeType{t}
}

var typeAliasTypeDefault = &TypeAliasType{`UnresolvedAlias`, nil, defaultTypeDefault, nil}
