package types

import (
	"fmt"
	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

var typeFunctionType = NewTypeType(DefaultCallableType())

const keyReturnsError = `returns_error`

var typeFunction = NewStructType([]*StructElement{
	newStructElement2(keyType, typeFunctionType),
	NewStructElement(newOptionalType3(keyFinal), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyOverride), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations),
	NewStructElement(newOptionalType3(KeyGoName), DefaultStringType()),
	NewStructElement(newOptionalType3(keyReturnsError), NewBooleanType(true)),
})

type function struct {
	annotatedMember
	returnsError bool
	goName       string
}

func newFunction(c px.Context, name string, container *objectType, initHash *Hash) px.ObjFunc {
	f := &function{}
	f.initialize(c, name, container, initHash)
	return f
}

func (f *function) initialize(c px.Context, name string, container *objectType, initHash *Hash) {
	px.AssertInstance(func() string { return fmt.Sprintf(`initializer function for %s[%s]`, container.Label(), name) }, typeFunction, initHash)
	f.annotatedMember.initialize(c, `function`, name, container, initHash)
	if gn, ok := initHash.Get4(KeyGoName); ok {
		f.goName = gn.String()
	}
	if re, ok := initHash.Get4(keyReturnsError); ok {
		f.returnsError = re.(booleanValue).Bool()
	}
}

func (f *function) Call(c px.Context, receiver px.Value, block px.Lambda, args []px.Value) px.Value {
	if f.CallableType().(*CallableType).CallableWith(args, block) {
		if co, ok := receiver.(px.CallableObject); ok {
			if result, ok := co.Call(c, f, args, block); ok {
				return result
			}
		}

		panic(px.Error(px.InstanceDoesNotRespond, issue.H{`type`: receiver.PType(), `message`: f.name}))
	}
	types := make([]px.Value, len(args))
	for i, a := range args {
		types[i] = a.PType()
	}
	panic(px.Error(px.TypeMismatch, issue.H{`detail`: px.DescribeSignatures(
		[]px.Signature{f.CallableType().(*CallableType)}, newTupleType2(types...), block)}))
}

func (f *function) CallGo(c px.Context, receiver interface{}, args ...interface{}) []interface{} {
	rfArgs := make([]reflect.Value, 1+len(args))
	rfArgs[0] = reflect.ValueOf(receiver)
	for i, arg := range args {
		rfArgs[i+1] = reflect.ValueOf(arg)
	}
	result := f.CallGoReflected(c, rfArgs)
	rs := make([]interface{}, len(result))
	for i, ret := range result {
		if ret.IsValid() {
			rs[i] = ret.Interface()
		}
	}
	return rs
}

func (f *function) CallGoReflected(c px.Context, args []reflect.Value) []reflect.Value {
	rt := args[0].Type()
	m, ok := rt.MethodByName(f.goName)
	if !ok {
		panic(px.Error(px.InstanceDoesNotRespond, issue.H{`type`: rt.String(), `message`: f.goName}))
	}

	mt := m.Type
	pc := mt.NumIn()
	if pc != len(args) {
		panic(px.Error(px.TypeMismatch, issue.H{`detail`: px.DescribeSignatures(
			[]px.Signature{f.CallableType().(*CallableType)}, NewTupleType([]px.Type{}, NewIntegerType(int64(pc-1), int64(pc-1))), nil)}))
	}
	result := m.Func.Call(args)
	oc := mt.NumOut()

	if f.ReturnsError() {
		oc--
		err := result[oc].Interface()
		if err != nil {
			if re, ok := err.(issue.Reported); ok {
				panic(re)
			}
			panic(px.Error(px.GoFunctionError, issue.H{`name`: f.goName, `error`: err}))
		}
		result = result[:oc]
	}
	return result
}

func (f *function) GoName() string {
	return f.goName
}

func (f *function) ReturnsError() bool {
	return f.returnsError
}

func (f *function) Equals(other interface{}, g px.Guard) bool {
	if of, ok := other.(*function); ok {
		return f.override == of.override && f.name == of.name && f.final == of.final && f.typ.Equals(of.typ, g)
	}
	return false
}

func (f *function) FeatureType() string {
	return `function`
}

func (f *function) Label() string {
	return fmt.Sprintf(`function %s[%s]`, f.container.Label(), f.Name())
}

func (f *function) CallableType() px.Type {
	return f.typ.(*CallableType)
}

func (f *function) InitHash() px.OrderedMap {
	return WrapStringPValue(f.initHash())
}
