package types

import (
	"io"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type InitType struct {
	typ      px.Type
	initArgs *Array
	ctor     px.Function
}

var InitMetaType px.ObjectType

func init() {
	InitMetaType = newObjectType(`Pcore::Init`, `Pcore::AnyType {
	attributes => {
    type => { type => Optional[Type], value => undef },
		init_args => { type => Array, value => [] }
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
		return newInitType2(args...)
	})
}

func DefaultInitType() *InitType {
	return initTypeDefault
}

func NewInitType(typ px.Value, args px.Value) *InitType {
	tp, ok := typ.(px.Type)
	if !ok {
		tp = nil
	}
	aa, ok := args.(*Array)
	if !ok {
		aa = emptyArray
	}
	if typ == nil && aa.Len() == 0 {
		return initTypeDefault
	}
	return &InitType{typ: tp, initArgs: aa}
}

func newInitType2(args ...px.Value) *InitType {
	switch len(args) {
	case 0:
		return DefaultInitType()
	case 1:
		return NewInitType(args[0], nil)
	default:
		return NewInitType(args[0], WrapValues(args[1:]))
	}
}

func (t *InitType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *InitType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *InitType) SerializationString() string {
	return t.String()
}

func (t *InitType) Default() px.Type {
	return initTypeDefault
}

func (t *InitType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*InitType); ok {
		return t == ot || px.Equals(t.typ, ot.typ, g) && px.Equals(t.initArgs, ot.initArgs, g)
	}
	return false
}

func (t *InitType) Get(key string) (px.Value, bool) {
	switch key {
	case `type`:
		if t.typ == nil {
			return undef, true
		}
		return t.typ, true
	case `init_args`:
		return t.initArgs, true
	default:
		return nil, false
	}
}

func (t *InitType) EachSignature(doer func(signature px.Signature)) {
	t.assertInitialized()
	if t.ctor != nil {
		for _, lambda := range t.ctor.Dispatchers() {
			doer(lambda.Signature())
		}
	}
}

func (t *InitType) anySignature(doer func(signature px.Signature) bool) bool {
	t.assertInitialized()
	if t.ctor != nil {
		for _, lambda := range t.ctor.Dispatchers() {
			if doer(lambda.Signature()) {
				return true
			}
		}
	}
	return false
}

// IsAssignable answers the question if a value of the given type can be used when
// instantiating an instance of the contained type
func (t *InitType) IsAssignable(o px.Type, g px.Guard) bool {
	if t.typ == nil {
		return richDataTypeDefault.IsAssignable(o, g)
	}

	if !t.initArgs.IsEmpty() {
		ts := append(make([]px.Type, 0, t.initArgs.Len()+1), o)
		t.initArgs.Each(func(v px.Value) { ts = append(ts, v.PType()) })
		tp := NewTupleType(ts, nil)
		return t.anySignature(func(s px.Signature) bool { return s.IsAssignable(tp, g) })
	}

	// First test if the given value matches a single value constructor
	tp := NewTupleType([]px.Type{o}, nil)
	return t.anySignature(func(s px.Signature) bool { return s.IsAssignable(tp, g) })
}

// IsInstance answers the question if the given value can be used when
// instantiating an instance of the contained type
func (t *InitType) IsInstance(o px.Value, g px.Guard) bool {
	if t.typ == nil {
		return richDataTypeDefault.IsInstance(o, g)
	}

	if !t.initArgs.IsEmpty() {
		// The init arguments must be combined with the given value in an array. Here, it doesn't
		// matter if the given value is an array or not. It must match as a single value regardless.
		vs := append(make([]px.Value, 0, t.initArgs.Len()+1), o)
		vs = t.initArgs.AppendTo(vs)
		return t.anySignature(func(s px.Signature) bool { return s.CallableWith(vs, nil) })
	}

	// First test if the given value matches a single value constructor.
	vs := []px.Value{o}
	if t.anySignature(func(s px.Signature) bool { return s.CallableWith(vs, nil) }) {
		return true
	}

	// If the given value is an array, expand it and check if it matches.
	if a, ok := o.(*Array); ok {
		vs = a.AppendTo(make([]px.Value, 0, a.Len()))
		return t.anySignature(func(s px.Signature) bool { return s.CallableWith(vs, nil) })
	}
	return false
}

func (t *InitType) MetaType() px.ObjectType {
	return InitMetaType
}

func (t *InitType) Name() string {
	return `Init`
}

func (t *InitType) New(c px.Context, args []px.Value) px.Value {
	t.Resolve(c)
	if t.ctor == nil {
		panic(px.Error(px.InstanceDoesNotRespond, issue.H{`type`: t, `message`: `new`}))
	}

	if !t.initArgs.IsEmpty() {
		// The init arguments must be combined with the given value in an array. Here, it doesn't
		// matter if the given value is an array or not. It must match as a single value regardless.
		vs := append(make([]px.Value, 0, t.initArgs.Len()+len(args)), args...)
		vs = t.initArgs.AppendTo(vs)
		return t.ctor.Call(c, nil, vs...)
	}

	// First test if the given value matches a single value constructor.
	if t.anySignature(func(s px.Signature) bool { return s.CallableWith(args, nil) }) {
		return t.ctor.Call(c, nil, args...)
	}

	// If the given value is an array, expand it and check if it matches.
	if len(args) == 1 {
		arg := args[0]
		if a, ok := arg.(*Array); ok {
			vs := a.AppendTo(make([]px.Value, 0, a.Len()))
			return t.ctor.Call(c, nil, vs...)
		}
	}

	// Provoke argument error
	return t.ctor.Call(c, nil, args...)
}

func (t *InitType) Resolve(c px.Context) px.Type {
	if t.typ != nil && t.ctor == nil {
		if ctor, ok := px.Load(c, NewTypedName(px.NsConstructor, t.typ.Name())); ok {
			t.ctor = ctor.(px.Function)
		} else {
			panic(px.Error(px.CtorNotFound, issue.H{`type`: t.typ.Name()}))
		}
	}
	return t
}

func (t *InitType) String() string {
	return px.ToString2(t, None)
}

func (t *InitType) Parameters() []px.Value {
	t.assertInitialized()
	if t.initArgs.Len() == 0 {
		if t.typ == nil {
			return px.EmptyValues
		}
		return []px.Value{t.typ}
	}
	ps := []px.Value{undef, t.initArgs}
	if t.typ != nil {
		ps[1] = t.typ
	}
	return ps
}

func (t *InitType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *InitType) Type() px.Type {
	return t.typ
}

func (t *InitType) PType() px.Type {
	return &TypeType{t}
}

func (t *InitType) assertInitialized() {
	if t.typ != nil && t.ctor == nil {
		t.Resolve(px.CurrentContext())
	}
}

var initTypeDefault = &InitType{typ: nil, initArgs: emptyArray}
