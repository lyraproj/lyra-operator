package types

import (
	"fmt"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
)

const KeyGoName = `go_name`

var typeAttributeKind = NewEnumType([]string{string(constant), string(derived), string(givenOrDerived), string(reference)}, false)

var typeAttribute = NewStructType([]*StructElement{
	newStructElement2(keyType, NewVariantType(DefaultTypeType(), TypeTypeName)),
	NewStructElement(newOptionalType3(keyFinal), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyOverride), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyKind), typeAttributeKind),
	NewStructElement(newOptionalType3(keyValue), DefaultAnyType()),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations),
	NewStructElement(newOptionalType3(KeyGoName), DefaultStringType()),
})

var typeAttributeCallable = newCallableType2(NewIntegerType(0, 0))

type attribute struct {
	annotatedMember
	kind   px.AttributeKind
	value  px.Value
	goName string
}

func newAttribute(c px.Context, name string, container *objectType, initHash *Hash) px.Attribute {
	a := &attribute{}
	a.initialize(c, name, container, initHash)
	return a
}

func (a *attribute) initialize(c px.Context, name string, container *objectType, initHash *Hash) {
	px.AssertInstance(func() string { return fmt.Sprintf(`initializer for attribute %s[%s]`, container.Label(), name) }, typeAttribute, initHash)
	a.annotatedMember.initialize(c, `attribute`, name, container, initHash)
	a.kind = px.AttributeKind(stringArg(initHash, keyKind, ``))
	if a.kind == constant { // final is implied
		if initHash.IncludesKey2(keyFinal) && !a.final {
			panic(px.Error(px.ConstantWithFinal, issue.H{`label`: a.Label()}))
		}
		a.final = true
	}
	v := initHash.Get5(keyValue, nil)
	if v != nil {
		if a.kind == derived || a.kind == givenOrDerived {
			panic(px.Error(px.IllegalKindValueCombination, issue.H{`label`: a.Label(), `kind`: a.kind}))
		}
		if _, ok := v.(*DefaultValue); ok || px.IsInstance(a.typ, v) {
			a.value = v
		} else {
			panic(px.Error(px.TypeMismatch, issue.H{`detail`: px.DescribeMismatch(a.Label(), a.typ, px.DetailedValueType(v))}))
		}
	} else {
		if a.kind == constant {
			panic(px.Error(px.ConstantRequiresValue, issue.H{`label`: a.Label()}))
		}
		if a.kind == givenOrDerived {
			// Type is always optional
			if !px.IsInstance(a.typ, undef) {
				a.typ = NewOptionalType(a.typ)
			}
		}
		if _, ok := a.typ.(*OptionalType); ok {
			a.value = undef // Optional attributes have an implicit value of undef
		} else {
			a.value = nil // Not to be confused with undef
		}
	}
	if gn, ok := initHash.Get4(KeyGoName); ok {
		a.goName = gn.String()
	}
}

func (a *attribute) Call(c px.Context, receiver px.Value, block px.Lambda, args []px.Value) px.Value {
	if block == nil && len(args) == 0 {
		return a.Get(receiver)
	}
	types := make([]px.Value, len(args))
	for i, a := range args {
		types[i] = a.PType()
	}
	panic(px.Error(px.TypeMismatch, issue.H{`detail`: px.DescribeSignatures(
		[]px.Signature{a.CallableType().(*CallableType)}, newTupleType2(types...), block)}))
}

func (a *attribute) Default(value px.Value) bool {
	return a.value != nil && a.value.Equals(value, nil)
}

func (a *attribute) GoName() string {
	return a.goName
}

func (a *attribute) Tags(c px.Context) px.TagsAnnotation {
	if ta, ok := a.Annotations(c).Get(TagsAnnotationType); ok {
		return ta.(px.TagsAnnotation)
	}
	return nil
}

func (a *attribute) Kind() px.AttributeKind {
	return a.kind
}

func (a *attribute) HasValue() bool {
	return a.value != nil
}

func (a *attribute) initHash() *hash.StringHash {
	h := a.annotatedMember.initHash()
	if a.kind != defaultKind {
		h.Put(keyKind, stringValue(string(a.kind)))
	}
	if a.value != nil {
		opt := a.value.Equals(undef, nil)
		if opt {
			_, opt = a.typ.(*OptionalType)
		}
		if !opt {
			h.Put(keyValue, a.value)
		}
	}
	return h
}

func (a *attribute) InitHash() px.OrderedMap {
	return WrapStringPValue(a.initHash())
}

func (a *attribute) Value() px.Value {
	if a.value == nil {
		panic(px.Error(px.AttributeHasNoValue, issue.H{`label`: a.Label()}))
	}
	return a.value
}

func (a *attribute) FeatureType() string {
	return `attribute`
}

func (a *attribute) Get(instance px.Value) px.Value {
	if a.kind == constant {
		return a.value
	}
	if v, ok := a.container.GetValue(a.name, instance); ok {
		return v
	}
	panic(px.Error(px.NoAttributeReader, issue.H{`label`: a.Label()}))
}

func (a *attribute) Label() string {
	return fmt.Sprintf(`attribute %s[%s]`, a.container.Label(), a.Name())
}

func (a *attribute) Equals(other interface{}, g px.Guard) bool {
	if oa, ok := other.(*attribute); ok {
		return a.kind == oa.kind && a.override == oa.override && a.name == oa.name && a.final == oa.final && a.typ.Equals(oa.typ, g)
	}
	return false
}

func (a *attribute) CallableType() px.Type {
	return typeAttributeCallable
}

func newTypeParameter(c px.Context, name string, container *objectType, initHash *Hash) px.Attribute {
	t := &typeParameter{}
	t.initialize(c, name, container, initHash)
	return t
}
