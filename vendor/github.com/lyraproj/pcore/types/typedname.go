package types

import (
	"io"
	"regexp"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type typedName struct {
	namespace px.Namespace
	authority px.URI
	name      string
	canonical string
	parts     []string
}

var TypedNameMetaType px.Type

func init() {
	TypedNameMetaType = newObjectType(`TypedName`, `{
    attributes => {
      'namespace' => String,
      'name' => String,
      'authority' => { type => Optional[URI], value => undef },
      'parts' => { type => Array[String], kind => derived },
      'is_qualified' => { type => Boolean, kind => derived },
      'child' => { type => Optional[TypedName], kind => derived },
      'parent' => { type => Optional[TypedName], kind => derived }
    },
    functions => {
      'is_parent' => Callable[[TypedName],Boolean],
      'relative_to' => Callable[[TypedName],Optional[TypedName]]
    }
  }`, func(ctx px.Context, args []px.Value) px.Value {
		ns := px.Namespace(args[0].String())
		n := args[1].String()
		if len(args) > 2 {
			return newTypedName2(ns, n, px.URI(args[2].(*UriValue).String()))
		}
		return NewTypedName(ns, n)
	}, func(ctx px.Context, args []px.Value) px.Value {
		h := args[0].(*Hash)
		ns := px.Namespace(h.Get5(`namespace`, px.EmptyString).String())
		n := h.Get5(`name`, px.EmptyString).String()
		if x, ok := h.Get4(`authority`); ok {
			return newTypedName2(ns, n, px.URI(x.(*UriValue).String()))
		}
		return NewTypedName(ns, n)
	})
}

func (t *typedName) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	ObjectToString(t, format, bld, g)
}

func (t *typedName) PType() px.Type {
	return TypedNameMetaType
}

func (t *typedName) Call(c px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	switch method.Name() {
	case `is_parent`:
		return booleanValue(t.IsParent(args[0].(px.TypedName))), true
	case `relative_to`:
		if r, ok := t.RelativeTo(args[0].(px.TypedName)); ok {
			return r, true
		}
		return undef, true
	}
	return nil, false
}

func (t *typedName) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `namespace`:
		return stringValue(string(t.namespace)), true
	case `authority`:
		if t.authority == px.RuntimeNameAuthority {
			return px.Undef, true
		}
		return WrapURI2(string(t.authority)), true
	case `name`:
		return stringValue(t.Name()), true
	case `parts`:
		return t.PartsList(), true
	case `is_qualified`:
		return booleanValue(t.IsQualified()), true
	case `parent`:
		p := t.Parent()
		if p == nil {
			return undef, true
		}
		return p, true
	case `child`:
		p := t.Child()
		if p == nil {
			return undef, true
		}
		return p, true
	}
	return nil, false
}

func (t *typedName) InitHash() px.OrderedMap {
	es := make([]*HashEntry, 0, 3)
	es = append(es, WrapHashEntry2(`namespace`, stringValue(string(t.Namespace()))))
	es = append(es, WrapHashEntry2(`name`, stringValue(t.Name())))
	if t.authority != px.RuntimeNameAuthority {
		es = append(es, WrapHashEntry2(`authority`, WrapURI2(string(t.authority))))
	}
	return WrapHash(es)
}

func NewTypedName(namespace px.Namespace, name string) px.TypedName {
	return newTypedName2(namespace, name, px.RuntimeNameAuthority)
}

var allowedCharacters = regexp.MustCompile(`\A[A-Za-z][0-9A-Z_a-z]*\z`)

func newTypedName2(namespace px.Namespace, name string, nameAuthority px.URI) px.TypedName {
	tn := typedName{}
	tn.namespace = namespace
	tn.authority = nameAuthority
	tn.name = strings.TrimPrefix(name, `::`)
	return &tn
}

func typedNameFromMapKey(mapKey string) px.TypedName {
	if i := strings.LastIndexByte(mapKey, '/'); i > 0 {
		pfx := mapKey[:i]
		name := mapKey[i+1:]
		if i = strings.LastIndexByte(pfx, '/'); i > 0 {
			return newTypedName2(px.Namespace(pfx[i+1:]), name, px.URI(pfx[:i]))
		}
	}
	panic(px.Error(px.InvalidTypedNameMapKey, issue.H{`mapKey`: mapKey}))
}

func (t *typedName) Child() px.TypedName {
	if !t.IsQualified() {
		return nil
	}
	return t.child(1)
}

func (t *typedName) child(stripCount int) px.TypedName {
	name := t.name
	sx := 0
	for i := 0; i < stripCount; i++ {
		sx = strings.Index(name, `::`)
		if sx < 0 {
			return nil
		}
		name = name[sx+2:]
	}

	tn := &typedName{
		namespace: t.namespace,
		authority: t.authority,
		name:      name}

	if t.canonical != `` {
		pfxLen := len(t.authority) + len(t.namespace) + 2
		diff := len(t.name) - len(name)
		tn.canonical = t.canonical[:pfxLen] + t.canonical[pfxLen+diff:]
	}
	if t.parts != nil {
		tn.parts = t.parts[stripCount:]
	}
	return tn
}

func (t *typedName) Parent() px.TypedName {
	lx := strings.LastIndex(t.name, `::`)
	if lx < 0 {
		return nil
	}
	tn := &typedName{
		namespace: t.namespace,
		authority: t.authority,
		name:      t.name[:lx]}

	if t.canonical != `` {
		pfxLen := len(t.authority) + len(t.namespace) + 2
		tn.canonical = t.canonical[:pfxLen+lx]
	}
	if t.parts != nil {
		tn.parts = t.parts[:len(t.parts)-1]
	}
	return tn
}

func (t *typedName) Equals(other interface{}, g px.Guard) bool {
	if tn, ok := other.(px.TypedName); ok {
		return t.MapKey() == tn.MapKey()
	}
	return false
}

func (t *typedName) Name() string {
	return t.name
}

func (t *typedName) IsParent(o px.TypedName) bool {
	tps := t.Parts()
	ops := o.Parts()
	top := len(tps)
	if top < len(ops) {
		for idx := 0; idx < top; idx++ {
			if tps[idx] != ops[idx] {
				return false
			}
		}
		return true
	}
	return false
}

func (t *typedName) RelativeTo(parent px.TypedName) (px.TypedName, bool) {
	if parent.IsParent(t) {
		return t.child(len(parent.Parts())), true
	}
	return nil, false
}

func (t *typedName) IsQualified() bool {
	if t.parts == nil {
		return strings.Contains(t.name, `::`)
	}
	return len(t.parts) > 1
}

func (t *typedName) MapKey() string {
	if t.canonical == `` {
		t.canonical = strings.ToLower(string(t.authority) + `/` + string(t.namespace) + `/` + t.name)
	}
	return t.canonical
}

func (t *typedName) Parts() []string {
	if t.parts == nil {
		parts := strings.Split(strings.ToLower(t.name), `::`)
		for _, part := range parts {
			if !allowedCharacters.MatchString(part) {
				panic(px.Error(px.InvalidCharactersInName, issue.H{`name`: t.name}))
			}
		}
		t.parts = parts
	}
	return t.parts
}

func (t *typedName) PartsList() px.List {
	parts := t.Parts()
	es := make([]px.Value, len(parts))
	for i, p := range parts {
		es[i] = stringValue(p)
	}
	return WrapValues(es)
}

func (t *typedName) String() string {
	return px.ToString(t)
}

func (t *typedName) Namespace() px.Namespace {
	return t.namespace
}

func (t *typedName) Authority() px.URI {
	return t.authority
}
