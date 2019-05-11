package types

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type typedObject struct {
	typ px.ObjectType
}

func (o *typedObject) PType() px.Type {
	return o.typ
}

func (o *typedObject) valuesFromHash(c px.Context, hash px.OrderedMap) []px.Value {
	typ := o.typ.(*objectType)
	va := typ.AttributesInfo().PositionalFromHash(hash)
	if len(va) > 0 && typ.IsParameterized() {
		params := make([]*HashEntry, 0)
		typ.typeParameters(true).EachPair(func(k string, v interface{}) {
			if pv, ok := hash.Get4(k); ok && px.IsInstance(v.(*typeParameter).typ, pv) {
				params = append(params, WrapHashEntry2(k, pv))
			}
		})
		if len(params) > 0 {
			o.typ = NewObjectTypeExtension(c, typ, []px.Value{WrapHash(params)})
		}
	}
	return va
}

type attributeSlice struct {
	typedObject
	values []px.Value
}

func AllocObjectValue(typ px.ObjectType) px.Object {
	if typ.IsMetaType() {
		return AllocObjectType()
	}
	if rf := typ.GoType(); rf != nil {
		if rf.Kind() == reflect.Ptr && rf.Elem().Kind() == reflect.Struct {
			rf = rf.Elem()
		}
		return &reflectedObject{typedObject{typ}, reflect.New(rf).Elem()}
	}
	return &attributeSlice{typedObject{typ}, px.EmptyValues}
}

func NewReflectedValue(typ px.ObjectType, value reflect.Value) px.Object {
	if value.Kind() == reflect.Func {
		return &reflectedFunc{typedObject{typ}, value}
	}
	return &reflectedObject{typedObject{typ}, value}
}

func NewObjectValue(c px.Context, typ px.ObjectType, values []px.Value) (ov px.Object) {
	ov = AllocObjectValue(typ)
	ov.Initialize(c, values)
	return ov
}

func newObjectValue2(c px.Context, typ px.ObjectType, hash *Hash) (ov px.Object) {
	ov = AllocObjectValue(typ)
	ov.InitFromHash(c, hash)
	return ov
}

func (o *attributeSlice) Reflect(c px.Context) reflect.Value {
	ot := o.PType().(px.ReflectedType)
	if v, ok := ot.ReflectType(c); ok {
		rv := reflect.New(v.Elem())
		o.ReflectTo(c, rv.Elem())
		return rv
	}
	panic(px.Error(px.UnreflectableValue, issue.H{`type`: o.PType()}))
}

func (o *attributeSlice) ReflectTo(c px.Context, value reflect.Value) {
	o.typ.ToReflectedValue(c, o, value)
}

func (o *attributeSlice) Initialize(c px.Context, values []px.Value) {
	if len(values) > 0 && o.typ.IsParameterized() {
		o.InitFromHash(c, makeValueHash(o.typ.AttributesInfo(), values))
		return
	}
	fillValueSlice(values, o.typ.AttributesInfo().Attributes())
	o.values = values
}

func (o *attributeSlice) InitFromHash(c px.Context, hash px.OrderedMap) {
	o.values = o.valuesFromHash(c, hash)
}

// Ensure that all entries in the value slice that are nil receive default values from the given attributes
func fillValueSlice(values []px.Value, attrs []px.Attribute) {
	for ix, v := range values {
		if v == nil {
			at := attrs[ix]
			if at.Kind() == givenOrDerived {
				values[ix] = undef
			} else {
				if !at.HasValue() {
					panic(px.Error(px.MissingRequiredAttribute, issue.H{`label`: at.Label()}))
				}
				values[ix] = at.Value()
			}
		}
	}
}

func (o *attributeSlice) Get(key string) (px.Value, bool) {
	pi := o.typ.AttributesInfo()
	if idx, ok := pi.NameToPos()[key]; ok {
		if idx < len(o.values) {
			return o.values[idx], ok
		}
		a := pi.Attributes()[idx]
		if a.Kind() == givenOrDerived {
			return undef, true
		}
		return a.Value(), ok
	}
	return nil, false
}

func (o *attributeSlice) Call(c px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	if v, ok := px.Load(c, NewTypedName(px.NsFunction, strings.ToLower(o.typ.Name())+`::`+method.Name())); ok {
		if f, ok := v.(px.Function); ok {
			return f.Call(c, block, args...), true
		}
	}
	return nil, false
}

func (o *attributeSlice) Equals(other interface{}, g px.Guard) bool {
	if ov, ok := other.(*attributeSlice); ok {
		return o.typ.Equals(ov.typ, g) && px.Equals(o.values, ov.values, g)
	}
	return false
}

func (o *attributeSlice) String() string {
	return px.ToString(o)
}

func (o *attributeSlice) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *attributeSlice) InitHash() px.OrderedMap {
	return makeValueHash(o.typ.AttributesInfo(), o.values)
}

// Turn a positional argument list into a hash. The hash will exclude all values
// that are equal to the default value of the corresponding attribute
func makeValueHash(pi px.AttributesInfo, values []px.Value) *Hash {
	at := pi.Attributes()
	entries := make([]*HashEntry, 0, len(at))
	for i, v := range values {
		attr := at[i]
		if !(attr.HasValue() && v.Equals(attr.Value(), nil) || attr.Kind() == givenOrDerived && v.Equals(undef, nil)) {
			entries = append(entries, WrapHashEntry2(attr.Name(), v))
		}
	}
	return WrapHash(entries)
}

type reflectedObject struct {
	typedObject
	value reflect.Value
}

func (o *reflectedObject) Call(c px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	m, ok := o.value.Type().MethodByName(method.GoName())
	if !ok {
		return nil, false
	}

	mt := m.Type
	rf := c.Reflector()
	var vat reflect.Type

	// argc, the number of arguments + the mandatory call receiver
	argc := len(args) + 1

	// number of expected arguments
	top := mt.NumIn()
	last := top - 1

	if mt.IsVariadic() {
		if argc < last {
			// Must be at least expected number of arguments minus one (variadic can have a zero count)
			panic(fmt.Errorf("argument count error. Expected at least %d, got %d", last, argc))
		}

		// Slice big enough to hold all variadics
		vat = mt.In(last).Elem()
	} else {
		if top != argc {
			panic(fmt.Errorf("argument count error. Expected %d, got %d", top, argc))
		}
	}

	rfArgs := make([]reflect.Value, argc)
	rfArgs[0] = o.value

	for i, arg := range args {
		pn := i + 1
		var tp reflect.Type
		if pn >= last && vat != nil {
			tp = vat
		} else {
			tp = mt.In(pn)
		}
		av := reflect.New(tp).Elem()
		rf.ReflectTo(arg, av)
		rfArgs[pn] = av
	}

	rr := method.(px.CallableGoMember).CallGoReflected(c, rfArgs)

	switch len(rr) {
	case 0:
		return undef, true
	case 1:
		r := rr[0]
		if r.IsValid() {
			return wrapReflected(c, r), true
		} else {
			return undef, true
		}
	default:
		rs := make([]px.Value, len(rr))
		for i, r := range rr {
			if r.IsValid() {
				rs[i] = wrapReflected(c, r)
			} else {
				rs[i] = undef
			}
		}
		return WrapValues(rs), true
	}
}

func (o *reflectedObject) Reflect(c px.Context) reflect.Value {
	return o.value
}

func (o *reflectedObject) ReflectTo(c px.Context, value reflect.Value) {
	if o.value.Kind() == reflect.Struct && value.Kind() == reflect.Ptr {
		value.Set(o.value.Addr())
	} else {
		value.Set(o.value)
	}
}

func (o *reflectedObject) Initialize(c px.Context, values []px.Value) {
	if len(values) > 0 && o.typ.IsParameterized() {
		o.InitFromHash(c, makeValueHash(o.typ.AttributesInfo(), values))
		return
	}
	pi := o.typ.AttributesInfo()
	attrs := pi.Attributes()
	if len(attrs) > 0 {
		attrs := pi.Attributes()
		fillValueSlice(values, attrs)
		o.setValues(c, values)
	} else if len(values) == 1 {
		values[0].(px.Reflected).ReflectTo(c, o.value)
	}
}

func (o *reflectedObject) InitFromHash(c px.Context, hash px.OrderedMap) {
	o.setValues(c, o.valuesFromHash(c, hash))
}

func (o *reflectedObject) setValues(c px.Context, values []px.Value) {
	attrs := o.typ.AttributesInfo().Attributes()
	rf := c.Reflector()
	if len(attrs) == 1 && attrs[0].GoName() == keyValue {
		rf.ReflectTo(values[0], o.value)
	} else {
		oe := o.structVal()
		for i, a := range attrs {
			var v px.Value
			if i < len(values) {
				v = values[i]
			} else {
				if a.HasValue() {
					v = a.Value()
				} else {
					v = undef
				}
			}
			rf.ReflectTo(v, oe.FieldByName(a.GoName()))
		}
	}
}

func (o *reflectedObject) Get(key string) (px.Value, bool) {
	pi := o.typ.AttributesInfo()
	if idx, ok := pi.NameToPos()[key]; ok {
		attr := pi.Attributes()[idx]
		if attr.GoName() == keyValue {
			return WrapPrimitive(o.value)
		}
		rf := o.structVal().FieldByName(attr.GoName())
		if rf.IsValid() {
			return wrap(nil, rf), true
		}
		a := pi.Attributes()[idx]
		if a.Kind() == givenOrDerived {
			return undef, true
		}
		return a.Value(), ok
	}
	return nil, false
}

func (o *reflectedObject) Equals(other interface{}, g px.Guard) bool {
	if o == other {
		return true
	}
	if ov, ok := other.(*reflectedObject); ok {
		for _, a := range o.typ.AttributesInfo().Attributes() {
			if !a.Get(o).Equals(a.Get(ov), g) {
				return false
			}
		}
		return true
	}
	return false
}

func (o *reflectedObject) String() string {
	return px.ToString(o)
}

func (o *reflectedObject) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *reflectedObject) InitHash() px.OrderedMap {
	pi := o.typ.AttributesInfo()
	at := pi.Attributes()
	nc := len(at)
	if nc == 0 {
		return px.EmptyMap
	}

	if nc == 1 {
		attr := at[0]
		if attr.GoName() == keyValue {
			pv, _ := WrapPrimitive(o.value)
			return singletonMap(`value`, pv)
		}
	}

	entries := make([]*HashEntry, 0, nc)
	oe := o.structVal()
	c := px.CurrentContext()
	for _, attr := range pi.Attributes() {
		gn := attr.GoName()
		if gn != `` {
			v := wrapReflected(c, oe.FieldByName(gn))
			if !(attr.HasValue() && v.Equals(attr.Value(), nil) || attr.Kind() == givenOrDerived && v.Equals(undef, nil)) {
				entries = append(entries, WrapHashEntry2(attr.Name(), v))
			}
		}
	}
	return WrapHash(entries)
}

func (o *reflectedObject) structVal() reflect.Value {
	oe := o.value
	if oe.Kind() == reflect.Ptr {
		oe = oe.Elem()
	}
	return oe
}

type reflectedFunc struct {
	typedObject
	function reflect.Value
}

func (o *reflectedFunc) Call(c px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	mt := o.function.Type()
	rf := c.Reflector()
	rfArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		av := reflect.New(mt.In(i)).Elem()
		rf.ReflectTo(arg, av)
		rfArgs[i] = av
	}

	pc := mt.NumIn()
	if pc != len(args) {
		panic(px.Error(px.TypeMismatch, issue.H{`detail`: px.DescribeSignatures(
			[]px.Signature{method.CallableType().(*CallableType)}, NewTupleType([]px.Type{}, NewIntegerType(int64(pc-1), int64(pc-1))), nil)}))
	}
	rr := o.function.Call(rfArgs)

	oc := mt.NumOut()

	if method.ReturnsError() {
		oc--
		err := rr[oc].Interface()
		if err != nil {
			if re, ok := err.(issue.Reported); ok {
				panic(re)
			}
			panic(px.Error(px.GoFunctionError, issue.H{`name`: mt.Name(), `error`: err}))
		}
		rr = rr[:oc]
	}

	switch len(rr) {
	case 0:
		return undef, true
	case 1:
		r := rr[0]
		if r.IsValid() {
			return wrap(c, r), true
		} else {
			return undef, true
		}
	default:
		rs := make([]px.Value, len(rr))
		for i, r := range rr {
			if r.IsValid() {
				rs[i] = wrap(c, r)
			} else {
				rs[i] = undef
			}
		}
		return WrapValues(rs), true
	}
}

func (o *reflectedFunc) Reflect(c px.Context) reflect.Value {
	return o.function
}

func (o *reflectedFunc) ReflectTo(c px.Context, value reflect.Value) {
	value.Set(o.function)
}

func (o *reflectedFunc) Initialize(c px.Context, arguments []px.Value) {
}

func (o *reflectedFunc) InitFromHash(c px.Context, hash px.OrderedMap) {
}

func (o *reflectedFunc) Get(key string) (px.Value, bool) {
	return nil, false
}

func (o *reflectedFunc) Equals(other interface{}, g px.Guard) bool {
	if ov, ok := other.(*reflectedFunc); ok {
		return o.typ.Equals(ov.typ, g) && o.function == ov.function
	}
	return false
}

func (o *reflectedFunc) String() string {
	return px.ToString(o)
}

func (o *reflectedFunc) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *reflectedFunc) InitHash() px.OrderedMap {
	return px.EmptyMap
}
