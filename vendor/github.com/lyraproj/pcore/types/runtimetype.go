package types

import (
	"fmt"
	"io"
	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type (
	RuntimeType struct {
		name    string
		runtime string
		pattern *RegexpType
		goType  reflect.Type
	}

	// RuntimeValue Captures values of all types unknown to Puppet
	RuntimeValue struct {
		puppetType *RuntimeType
		value      interface{}
	}
)

var runtimeTypeDefault = &RuntimeType{``, ``, nil, nil}

var RuntimeMetaType px.ObjectType

func init() {
	RuntimeMetaType = newObjectType(`Pcore::RuntimeType`,
		`Pcore::AnyType {
	attributes => {
		runtime => {
      type => Optional[String[1]],
      value => undef
    },
		name_or_pattern => {
      type => Variant[Undef,String[1],Tuple[Regexp,String[1]]],
      value => undef
    }
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newRuntimeType2(args...)
		})
}

func DefaultRuntimeType() *RuntimeType {
	return runtimeTypeDefault
}

func NewRuntimeType(runtimeName string, name string, pattern *RegexpType) *RuntimeType {
	if runtimeName == `` && name == `` && pattern == nil {
		return DefaultRuntimeType()
	}
	if runtimeName == `go` && name != `` {
		panic(px.Error(px.GoRuntimeTypeWithoutGoType, issue.H{`name`: name}))
	}
	return &RuntimeType{runtime: runtimeName, name: name, pattern: pattern}
}

func newRuntimeType2(args ...px.Value) *RuntimeType {
	top := len(args)
	if top > 3 {
		panic(illegalArgumentCount(`Runtime[]`, `0 - 3`, len(args)))
	}
	if top == 0 {
		return DefaultRuntimeType()
	}

	runtimeName, ok := args[0].(stringValue)
	if !ok {
		panic(illegalArgumentType(`Runtime[]`, 0, `String`, args[0]))
	}

	var pattern *RegexpType
	var name px.Value
	if top == 1 {
		name = px.EmptyString
	} else {
		var rv stringValue
		rv, ok = args[1].(stringValue)
		if !ok {
			panic(illegalArgumentType(`Runtime[]`, 1, `String`, args[1]))
		}
		name = rv

		if top == 2 {
			pattern = nil
		} else {
			pattern, ok = args[2].(*RegexpType)
			if !ok {
				panic(illegalArgumentType(`Runtime[]`, 2, `Type[Regexp]`, args[2]))
			}
		}
	}
	return NewRuntimeType(string(runtimeName), name.String(), pattern)
}

// NewGoRuntimeType creates a Go runtime by extracting the element type of the given value.
//
// A Go interface must be registered by passing a Pointer to a nil of the interface so to
// create an Runtime for the interface Foo, use NewGoRuntimeType((*Foo)(nil))
func NewGoRuntimeType(value interface{}) *RuntimeType {
	goType := reflect.TypeOf(value)
	if goType.Kind() == reflect.Ptr && goType.Elem().Kind() == reflect.Interface {
		goType = goType.Elem()
	}
	return &RuntimeType{runtime: `go`, name: goType.String(), goType: goType}
}

func (t *RuntimeType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *RuntimeType) Default() px.Type {
	return runtimeTypeDefault
}

func (t *RuntimeType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*RuntimeType); ok && t.runtime == ot.runtime && t.name == ot.name {
		if t.pattern == nil {
			return ot.pattern == nil
		}
		return t.pattern.Equals(ot.pattern, g)
	}
	return false
}

func (t *RuntimeType) Generic() px.Type {
	return runtimeTypeDefault
}

func (t *RuntimeType) Get(key string) (px.Value, bool) {
	switch key {
	case `runtime`:
		if t.runtime == `` {
			return undef, true
		}
		return stringValue(t.runtime), true
	case `name_or_pattern`:
		if t.pattern != nil {
			return t.pattern, true
		}
		if t.name != `` {
			return stringValue(t.name), true
		}
		return undef, true
	default:
		return nil, false
	}
}

func (t *RuntimeType) IsAssignable(o px.Type, g px.Guard) bool {
	if rt, ok := o.(*RuntimeType); ok {
		if t.goType != nil && rt.goType != nil {
			return rt.goType.AssignableTo(t.goType)
		}
		if t.runtime == `` {
			return true
		}
		if t.runtime != rt.runtime {
			return false
		}
		if t.name == `` {
			return true
		}
		if t.pattern != nil {
			return t.name == rt.name && rt.pattern != nil && t.pattern.pattern.String() == rt.pattern.pattern.String()
		}
		if t.name == rt.name {
			return true
		}
	}
	return false
}

func (t *RuntimeType) IsInstance(o px.Value, g px.Guard) bool {
	rt, ok := o.(*RuntimeValue)
	if !ok {
		return false
	}
	if t.goType != nil {
		return reflect.ValueOf(rt.Interface()).Type().AssignableTo(t.goType)
	}
	if t.runtime == `` {
		return true
	}
	if o == nil || t.runtime != `go` || t.pattern != nil {
		return false
	}
	if t.name == `` {
		return true
	}
	return t.name == fmt.Sprintf(`%T`, rt.Interface())
}

func (t *RuntimeType) MetaType() px.ObjectType {
	return RuntimeMetaType
}

func (t *RuntimeType) Name() string {
	return `Runtime`
}

func (t *RuntimeType) Parameters() []px.Value {
	if t.runtime == `` {
		return px.EmptyValues
	}
	ps := make([]px.Value, 0, 2)
	ps = append(ps, stringValue(t.runtime))
	if t.name != `` {
		ps = append(ps, stringValue(t.name))
	}
	if t.pattern != nil {
		ps = append(ps, t.pattern)
	}
	return ps
}

func (t *RuntimeType) ReflectType(c px.Context) (reflect.Type, bool) {
	return t.goType, t.goType != nil
}

func (t *RuntimeType) CanSerializeAsString() bool {
	return true
}

func (t *RuntimeType) SerializationString() string {
	return t.String()
}

func (t *RuntimeType) String() string {
	return px.ToString2(t, None)
}

func (t *RuntimeType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *RuntimeType) PType() px.Type {
	return &TypeType{t}
}

func WrapRuntime(value interface{}) *RuntimeValue {
	goType := reflect.TypeOf(value)
	return &RuntimeValue{&RuntimeType{runtime: `go`, name: goType.String(), goType: goType}, value}
}

func (rv *RuntimeValue) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*RuntimeValue); ok {
		var re px.Equality
		if re, ok = rv.value.(px.Equality); ok {
			var oe px.Equality
			if oe, ok = ov.value.(px.Equality); ok {
				return re.Equals(oe, g)
			}
			return false
		}
		return reflect.DeepEqual(rv.value, ov.value)
	}
	return false
}

func (rv *RuntimeValue) Reflect(c px.Context) reflect.Value {
	gt := rv.puppetType.goType
	if gt == nil {
		panic(px.Error(px.InvalidSourceForGet, issue.H{`type`: rv.PType().String()}))
	}
	return reflect.ValueOf(rv.value)
}

func (rv *RuntimeValue) ReflectTo(c px.Context, dest reflect.Value) {
	gt := rv.puppetType.goType
	if gt == nil {
		panic(px.Error(px.InvalidSourceForGet, issue.H{`type`: rv.PType().String()}))
	}
	if !gt.AssignableTo(dest.Type()) {
		panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: gt.String(), `actual`: dest.Type().String()}))
	}
	dest.Set(reflect.ValueOf(rv.value))
}

func (rv *RuntimeValue) String() string {
	return px.ToString2(rv, None)
}

func (rv *RuntimeValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	utils.PuppetQuote(b, fmt.Sprintf(`%v`, rv.value))
}

func (rv *RuntimeValue) PType() px.Type {
	return rv.puppetType
}

func (rv *RuntimeValue) Interface() interface{} {
	return rv.value
}
