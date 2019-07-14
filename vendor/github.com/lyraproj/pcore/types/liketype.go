package types

import (
	"io"

	"strconv"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type LikeType struct {
	baseType   px.Type
	resolved   px.Type
	navigation string
}

var LikeMetaType px.ObjectType

func init() {
	LikeMetaType = newObjectType(`Pcore::Like`,
		`Pcore::AnyType {
	attributes => {
    base_type => Type,
		navigation => String[1]
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newLikeType2(args...)
		})
}

func DefaultLikeType() *LikeType {
	return typeOfTypeDefault
}

func NewLikeType(baseType px.Type, navigation string) *LikeType {
	return &LikeType{baseType: baseType, navigation: navigation}
}

func newLikeType2(args ...px.Value) *LikeType {
	switch len(args) {
	case 0:
		return DefaultLikeType()
	case 2:
		if tp, ok := args[0].(px.Type); ok {
			if an, ok := args[1].(stringValue); ok {
				return NewLikeType(tp, string(an))
			} else {
				panic(illegalArgumentType(`Like[]`, 1, `String`, args[1]))
			}
		} else {
			panic(illegalArgumentType(`Like[]`, 0, `Type`, args[1]))
		}
	default:
		panic(illegalArgumentCount(`Like[]`, `0 or 2`, len(args)))
	}
}

func (t *LikeType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.baseType.Accept(v, g)
}

func (t *LikeType) Default() px.Type {
	return typeOfTypeDefault
}

func (t *LikeType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*LikeType); ok {
		return t.navigation == ot.navigation && t.baseType.Equals(ot.baseType, g)
	}
	return false
}

func (t *LikeType) Get(key string) (px.Value, bool) {
	switch key {
	case `base_type`:
		return t.baseType, true
	case `navigation`:
		return stringValue(t.navigation), true
	default:
		return nil, false
	}
}

func (t *LikeType) IsAssignable(o px.Type, g px.Guard) bool {
	return t.Resolve(nil).IsAssignable(o, g)
}

func (t *LikeType) IsInstance(o px.Value, g px.Guard) bool {
	return t.Resolve(nil).IsInstance(o, g)
}

func (t *LikeType) MetaType() px.ObjectType {
	return LikeMetaType
}

func (t *LikeType) Name() string {
	return `Like`
}

func (t *LikeType) String() string {
	return px.ToString2(t, None)
}

func (t *LikeType) Parameters() []px.Value {
	if *t == *typeOfTypeDefault {
		return px.EmptyValues
	}
	return []px.Value{t.baseType, stringValue(t.navigation)}
}

func (t *LikeType) Resolve(c px.Context) px.Type {
	if t.resolved != nil {
		return t.resolved
	}
	bt := t.baseType
	bv := bt.(px.Value)
	var ok bool
	for _, part := range strings.Split(t.navigation, `.`) {
		if c, bv, ok = navigate(c, bv, part); !ok {
			panic(px.Error(px.UnresolvedTypeOf, issue.H{`type`: t.baseType, `navigation`: t.navigation}))
		}
	}
	if bt, ok = bv.(px.Type); ok {
		t.resolved = bt
		return bt
	}
	panic(px.Error(px.UnresolvedTypeOf, issue.H{`type`: t.baseType, `navigation`: t.navigation}))
}

func (t *LikeType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *LikeType) PType() px.Type {
	return &TypeType{t}
}

func navigate(c px.Context, value px.Value, member string) (px.Context, px.Value, bool) {
	if typ, ok := value.(px.Type); ok {
		if po, ok := typ.(px.TypeWithCallableMembers); ok {
			if m, ok := po.Member(member); ok {
				if a, ok := m.(px.Attribute); ok {
					return c, a.Type(), true
				}
				if f, ok := m.(px.Function); ok {
					return c, f.PType().(*CallableType).ReturnType(), true
				}
			}
		} else if st, ok := typ.(*StructType); ok {
			if m, ok := st.HashedMembers()[member]; ok {
				return c, m.Value(), true
			}
		} else if tt, ok := typ.(*TupleType); ok {
			if n, err := strconv.ParseInt(member, 0, 64); err == nil {
				if et, ok := tt.At(int(n)).(px.Type); ok {
					return c, et, true
				}
			}
		} else if ta, ok := typ.(*TypeAliasType); ok {
			return navigate(c, ta.ResolvedType(), member)
		} else {
			if m, ok := typ.MetaType().Member(member); ok {
				if c == nil {
					c = px.CurrentContext()
				}
				return c, m.Call(c, typ, nil, []px.Value{}), true
			}
		}
	} else {
		if po, ok := value.PType().(px.TypeWithCallableMembers); ok {
			if m, ok := po.Member(member); ok {
				if c == nil {
					c = px.CurrentContext()
				}
				return c, m.Call(c, value, nil, []px.Value{}), true
			}
		}
	}
	return c, nil, false
}

var typeOfTypeDefault = &LikeType{baseType: DefaultAnyType()}
