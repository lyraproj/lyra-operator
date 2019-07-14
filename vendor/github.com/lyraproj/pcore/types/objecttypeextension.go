package types

import (
	"io"

	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
)

type objectTypeExtension struct {
	baseType   *objectType
	parameters *hash.StringHash
}

var ObjectTypeExtensionMetaType px.ObjectType

func init() {
	ObjectTypeExtensionMetaType = newObjectType(`Pcore::ObjectTypeExtensionType`,
		`Pcore::AnyType {
			attributes => {
				base_type => Type,
				init_parameters => Array
			}
		}`)
}

func NewObjectTypeExtension(c px.Context, baseType px.ObjectType, initParameters []px.Value) *objectTypeExtension {
	o := &objectTypeExtension{}
	o.initialize(c, baseType.(*objectType), initParameters)
	return o
}

func (te *objectTypeExtension) Accept(v px.Visitor, g px.Guard) {
	v(te)
	te.baseType.Accept(v, g)
}

func (te *objectTypeExtension) Annotations(c px.Context) px.OrderedMap {
	return te.baseType.Annotations(c)
}

func (te *objectTypeExtension) Constructor(c px.Context) px.Function {
	return te.baseType.Constructor(c)
}

func (te *objectTypeExtension) Default() px.Type {
	return te.baseType.Default()
}

func (te *objectTypeExtension) Equals(other interface{}, g px.Guard) bool {
	op, ok := other.(*objectTypeExtension)
	return ok && te.baseType.Equals(op.baseType, g) && te.parameters.Equals(op.parameters, g)
}

func (te *objectTypeExtension) Functions(includeParent bool) []px.ObjFunc {
	return te.baseType.Functions(includeParent)
}

func (te *objectTypeExtension) Generalize() px.Type {
	return te.baseType
}

func (te *objectTypeExtension) Get(key string) (px.Value, bool) {
	return te.baseType.Get(key)
}

func (te *objectTypeExtension) GoType() reflect.Type {
	return te.baseType.GoType()
}

func (te *objectTypeExtension) HasHashConstructor() bool {
	return te.baseType.HasHashConstructor()
}

func (te *objectTypeExtension) Implements(ifd px.ObjectType, g px.Guard) bool {
	return te.baseType.Implements(ifd, g)
}

func (te *objectTypeExtension) IsInterface() bool {
	return false
}

func (te *objectTypeExtension) IsAssignable(t px.Type, g px.Guard) bool {
	if ote, ok := t.(*objectTypeExtension); ok {
		return te.baseType.IsAssignable(ote.baseType, g) && te.testAssignable(ote.parameters, g)
	}
	if ot, ok := t.(*objectType); ok {
		return te.baseType.IsAssignable(ot, g) && te.testAssignable(hash.EmptyStringHash, g)
	}
	return false
}

func (te *objectTypeExtension) IsMetaType() bool {
	return te.baseType.IsMetaType()
}

func (te *objectTypeExtension) IsParameterized() bool {
	return true
}

func (te *objectTypeExtension) IsInstance(v px.Value, g px.Guard) bool {
	return te.baseType.IsInstance(v, g) && te.testInstance(v, g)
}

func (te *objectTypeExtension) Member(name string) (px.CallableMember, bool) {
	return te.baseType.Member(name)
}

func (te *objectTypeExtension) MetaType() px.ObjectType {
	return ObjectTypeExtensionMetaType
}

func (te *objectTypeExtension) Name() string {
	return te.baseType.Name()
}

func (te *objectTypeExtension) Parameters() []px.Value {
	pts := te.baseType.typeParameters(true)
	n := pts.Len()
	if n > 2 {
		return []px.Value{WrapStringPValue(te.parameters)}
	}
	params := make([]px.Value, 0, n)
	top := 0
	idx := 0
	pts.EachKey(func(k string) {
		v, ok := te.parameters.Get3(k)
		if ok {
			top = idx + 1
		} else {
			v = WrapDefault()
		}
		params = append(params, v.(px.Value))
		idx++
	})
	return params[:top]
}

func (te *objectTypeExtension) FromReflectedValue(c px.Context, src reflect.Value) px.PuppetObject {
	return te.baseType.FromReflectedValue(c, src)
}

func (te *objectTypeExtension) Parent() px.Type {
	return te.baseType.Parent()
}

func (te *objectTypeExtension) ToReflectedValue(c px.Context, src px.PuppetObject, dest reflect.Value) {
	te.baseType.ToReflectedValue(c, src, dest)
}

func (te *objectTypeExtension) String() string {
	return px.ToString2(te, None)
}

func (te *objectTypeExtension) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	TypeToString(te, bld, format, g)
}

func (te *objectTypeExtension) PType() px.Type {
	return &TypeType{te}
}

func (te *objectTypeExtension) initialize(c px.Context, baseType *objectType, initParameters []px.Value) {
	pts := baseType.typeParameters(true)
	pvs := pts.Values()
	if pts.IsEmpty() {
		panic(px.Error(px.NotParameterizedType, issue.H{`type`: baseType.Label()}))
	}
	te.baseType = baseType
	namedArgs := false
	if len(initParameters) == 1 {
		_, namedArgs = initParameters[0].(*Hash)
	}

	if namedArgs {
		namedArgs = pts.Len() >= 1 && !px.IsInstance(pvs[0].(*typeParameter).Type(), initParameters[0])
	}

	checkParam := func(tp *typeParameter, v px.Value) px.Value {
		return px.AssertInstance(func() string { return tp.Label() }, tp.Type(), v)
	}

	byName := hash.NewStringHash(pts.Len())
	if namedArgs {
		h := initParameters[0].(*Hash)
		h.EachPair(func(k, pv px.Value) {
			pn := k.String()
			tp := pts.Get(pn, nil)
			if tp == nil {
				panic(px.Error(px.MissingTypeParameter, issue.H{`name`: pn, `label`: baseType.Label()}))
			}
			if !pv.Equals(WrapDefault(), nil) {
				byName.Put(pn, checkParam(tp.(*typeParameter), pv))
			}
		})
	} else {
		for idx, t := range pvs {
			if idx < len(initParameters) {
				tp := t.(*typeParameter)
				pv := initParameters[idx]
				if !pv.Equals(WrapDefault(), nil) {
					byName.Put(tp.Name(), checkParam(tp, pv))
				}
			}
		}
	}
	if byName.IsEmpty() {
		panic(px.Error(px.EmptyTypeParameterList, issue.H{`label`: baseType.Label()}))
	}
	te.parameters = byName
}

func (te *objectTypeExtension) AttributesInfo() px.AttributesInfo {
	return te.baseType.AttributesInfo()
}

// Checks that the given `paramValues` hash contains all keys present in the `parameters` of
// this instance and that each keyed value is a match for the given parameter. The match is done
// using case expression semantics.
//
// This method is only called when a given type is found to be assignable to the base type of
// this extension.
func (te *objectTypeExtension) testAssignable(paramValues *hash.StringHash, g px.Guard) bool {
	// Default implementation performs case expression style matching of all parameter values
	// provided that the value exist (this should always be the case, since all defaults have
	// been assigned at this point)
	return te.parameters.AllPair(func(key string, v1 interface{}) bool {
		if v2, ok := paramValues.Get3(key); ok {
			a := v2.(px.Value)
			b := v1.(px.Value)
			if px.PuppetMatch(a, b) {
				return true
			}
			if at, ok := a.(px.Type); ok {
				if bt, ok := b.(px.Type); ok {
					return px.IsAssignable(bt, at)
				}
			}
		}
		return false
	})
}

// Checks that the given instance `o` has one attribute for each key present in the `parameters` of
// this instance and that each attribute value is a match for the given parameter. The match is done
// using case expression semantics.
//
// This method is only called when the given value is found to be an instance of the base type of
// this extension.
func (te *objectTypeExtension) testInstance(o px.Value, g px.Guard) bool {
	return te.parameters.AllPair(func(key string, v1 interface{}) bool {
		v2, ok := te.baseType.GetValue(key, o)
		return ok && px.PuppetMatch(v2, v1.(px.Value))
	})
}
