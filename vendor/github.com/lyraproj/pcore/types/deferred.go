package types

import (
	"io"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/pcore/px"
)

var DeferredMetaType px.ObjectType

func init() {
	DeferredMetaType = newObjectType(`Deferred`, `{
    attributes => {
      # Fully qualified name of the function
      name  => { type => Pattern[/\A[$]?[a-z][0-9A-Za-z_]*(?:::[a-z][0-9A-Za-z_]*)*\z/] },
      arguments => { type => Optional[Array[Any]], value => undef},
    }}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return newDeferred2(args...)
		},
		func(ctx px.Context, args []px.Value) px.Value {
			return newDeferredFromHash(args[0].(*Hash))
		})
}

type Deferred interface {
	px.Value

	Name() string

	Arguments() *Array

	Resolve(c px.Context, scope px.Keyed) px.Value
}

type deferred struct {
	name      string
	arguments *Array
}

func NewDeferred(name string, arguments ...px.Value) *deferred {
	return &deferred{name, WrapValues(arguments)}
}

func newDeferred2(args ...px.Value) *deferred {
	argc := len(args)
	if argc < 1 || argc > 2 {
		panic(illegalArgumentCount(`Deferred[]`, `1 - 2`, argc))
	}
	if name, ok := args[0].(stringValue); ok {
		if argc == 1 {
			return &deferred{string(name), emptyArray}
		}
		if as, ok := args[1].(*Array); ok {
			return &deferred{string(name), as}
		}
		panic(illegalArgumentType(`deferred[]`, 1, `Array`, args[1]))
	}
	panic(illegalArgumentType(`deferred[]`, 0, `String`, args[0]))
}

func newDeferredFromHash(hash *Hash) *deferred {
	name := hash.Get5(`name`, px.EmptyString).String()
	arguments := hash.Get5(`arguments`, px.EmptyArray).(*Array)
	return &deferred{name, arguments}
}

func (e *deferred) Name() string {
	return e.name
}

func (e *deferred) Arguments() *Array {
	return e.arguments
}

func (e *deferred) String() string {
	return px.ToString(e)
}

func (e *deferred) Equals(other interface{}, guard px.Guard) bool {
	if o, ok := other.(*deferred); ok {
		return e.name == o.name &&
			px.Equals(e.arguments, o.arguments, guard)
	}
	return false
}

func (e *deferred) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	ObjectToString(e, s, b, g)
}

func (e *deferred) PType() px.Type {
	return DeferredMetaType
}

func (e *deferred) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `name`:
		return stringValue(e.name), true
	case `arguments`:
		return e.arguments, true
	}
	return nil, false
}

func (e *deferred) InitHash() px.OrderedMap {
	return WrapHash([]*HashEntry{WrapHashEntry2(`name`, stringValue(e.name)), WrapHashEntry2(`arguments`, e.arguments)})
}

func (e *deferred) Resolve(c px.Context, scope px.Keyed) px.Value {
	fn := e.Name()
	da := e.Arguments()

	var args []px.Value
	if fn[0] == '$' {
		vn := fn[1:]
		vv, ok := scope.Get(stringValue(vn))
		if !ok {
			panic(px.Error(px.UnknownVariable, issue.H{`name`: vn}))
		}
		if da.Len() == 0 {
			// No point digging with zero arguments
			return vv
		}
		fn = `dig`
		args = append(make([]px.Value, 0, 1+da.Len()), vv)
	} else {
		args = make([]px.Value, 0, da.Len())
	}
	args = da.AppendTo(args)
	for i, a := range args {
		args[i] = ResolveDeferred(c, a, scope)
	}
	return px.Call(c, fn, args, nil)
}

// ResolveDeferred will resolve all occurrences of a DeferredValue in its
// given argument. Array and Hash arguments will be resolved recursively.
func ResolveDeferred(c px.Context, a px.Value, scope px.Keyed) px.Value {
	switch a := a.(type) {
	case Deferred:
		return a.Resolve(c, scope)
	case *DeferredType:
		return a.Resolve(c)
	case *Array:
		return a.Map(func(v px.Value) px.Value {
			return ResolveDeferred(c, v, scope)
		})
	case *Hash:
		return a.MapEntries(func(v px.MapEntry) px.MapEntry {
			return WrapHashEntry(ResolveDeferred(c, v.Key(), scope), ResolveDeferred(c, v.Value(), scope))
		})
	default:
		return a
	}
}
