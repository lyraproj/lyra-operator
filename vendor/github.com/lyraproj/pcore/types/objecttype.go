package types

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sync/atomic"

	"bytes"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

// TODO: Implement Type.CanCoerce(Value)
//   Object.IsInstance is true if value is a Hash matching the attribute requirements
//   where each entry value can be coerced into the attribute´s type
var ObjectMetaType px.ObjectType

func init() {
	oneArgCtor := func(ctx px.Context, args []px.Value) px.Value {
		return newObjectType2(ctx, args...)
	}
	ObjectMetaType = MakeObjectType(`Pcore::ObjectType`, AnyMetaType,
		WrapStringToValueMap(map[string]px.Value{
			`attributes`: singletonMap(`_pcore_init_hash`, TypeObjectInitHash)}), true,
		oneArgCtor, oneArgCtor)
}

const (
	keyAnnotations         = `annotations`
	keyAttributes          = `attributes`
	keyConstants           = `constants`
	keyEquality            = `equality`
	keyEqualityIncludeType = `equality_include_type`
	keyFinal               = `final`
	keyFunctions           = `functions`
	keyKind                = `kind`
	keyName                = `name`
	keyOverride            = `override`
	keyParent              = `parent`
	keySerialization       = `serialization`
	keyType                = `type`
	keyTypeParameters      = `type_parameters`
	keyValue               = `value`

	defaultKind    = px.AttributeKind(``)
	constant       = px.AttributeKind(`constant`)
	derived        = px.AttributeKind(`derived`)
	givenOrDerived = px.AttributeKind(`given_or_derived`)
	reference      = px.AttributeKind(`reference`)
)

var TypeNamePattern = regexp.MustCompile(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)

var TypeTypeName = NewPatternType([]*RegexpType{NewRegexpTypeR(TypeNamePattern)})

var MemberNamePattern = regexp.MustCompile(`\A[a-z_]\w*\z`)

var TypeMemberName = newPatternType2(NewRegexpTypeR(MemberNamePattern))

var TypeMemberNames = newArrayType2(TypeMemberName)
var TypeParameters = NewHashType(TypeMemberName, DefaultNotUndefType(), nil)
var TypeAttributes = NewHashType(TypeMemberName, DefaultNotUndefType(), nil)
var TypeConstants = NewHashType(TypeMemberName, DefaultAnyType(), nil)
var TypeFunctions = NewHashType(newVariantType2(TypeMemberName, newPatternType2(NewRegexpTypeR(regexp.MustCompile(`^\[]$`)))), DefaultNotUndefType(), nil)
var TypeEquality = newVariantType2(TypeMemberName, TypeMemberNames)

var TypeObjectInitHash = NewStructType([]*StructElement{
	NewStructElement(newOptionalType3(keyName), TypeTypeName),
	NewStructElement(newOptionalType3(keyParent), NewVariantType(DefaultTypeType(), TypeTypeName)),
	NewStructElement(newOptionalType3(keyTypeParameters), TypeParameters),
	NewStructElement(newOptionalType3(keyAttributes), TypeAttributes),
	NewStructElement(newOptionalType3(keyConstants), TypeConstants),
	NewStructElement(newOptionalType3(keyFunctions), TypeFunctions),
	NewStructElement(newOptionalType3(keyEquality), TypeEquality),
	NewStructElement(newOptionalType3(keyEqualityIncludeType), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyEquality), TypeEquality),
	NewStructElement(newOptionalType3(keySerialization), TypeMemberNames),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations),
})

type objectType struct {
	annotatable
	hashKey             px.HashKey
	name                string
	parent              px.Type
	creators            []px.DispatchFunction
	parameters          *hash.StringHash // map doesn't preserve order
	attributes          *hash.StringHash
	functions           *hash.StringHash
	equality            []string
	equalityIncludeType bool
	serialization       []string
	loader              px.Loader
	initHashExpression  interface{} // Expression, *Hash, or Go zero value
	attrInfo            *attributesInfo
	ctor                px.Function
	goType              px.AnnotatedType
	isInterface         bool
	initType            *StructType
}

func (t *objectType) ReflectType(c px.Context) (reflect.Type, bool) {
	if t.goType != nil {
		return t.goType.Type(), true
	}
	return c.ImplementationRegistry().TypeToReflected(t)
}

func ObjectToString(o px.PuppetObject, s px.FormatContext, b io.Writer, g px.RDetect) {
	indent := s.Indentation()
	if indent.Breaks() {
		utils.WriteString(b, "\n")
		utils.WriteString(b, indent.Padding())
	}
	n := o.PType().Name()
	ih := o.InitHash().(*Hash)
	if n == `` {
		// Anonymous objects can't be written in constructor call form. They are instead
		// written as a Hash
		ih.ToString(b, s, g)
	} else {
		utils.WriteString(b, n)
		ih.ToString2(b, s, px.GetFormat(s.FormatMap(), o.PType()), '(', g)
	}
}

var objectTypeDefault = &objectType{
	annotatable:         annotatable{annotations: emptyMap},
	name:                `Object`,
	hashKey:             px.HashKey("\x00tObject"),
	parameters:          hash.EmptyStringHash,
	attributes:          hash.EmptyStringHash,
	functions:           hash.EmptyStringHash,
	equalityIncludeType: true}

func DefaultObjectType() *objectType {
	return objectTypeDefault
}

var objectId = int64(0)

func AllocObjectType() *objectType {
	return &objectType{
		annotatable:         annotatable{annotations: emptyMap},
		hashKey:             px.HashKey(fmt.Sprintf("\x00tObject%d", atomic.AddInt64(&objectId, 1))),
		parameters:          hash.EmptyStringHash,
		attributes:          hash.EmptyStringHash,
		functions:           hash.EmptyStringHash,
		equalityIncludeType: true}
}

func BuildObjectType(name string, parent px.Type, hashProducer func(px.ObjectType) px.OrderedMap) px.ObjectType {
	obj := AllocObjectType()
	obj.name = name
	obj.parent = parent
	obj.initHashExpression = hashProducer(obj)
	return obj
}

func (t *objectType) Initialize(c px.Context, args []px.Value) {
	if len(args) == 1 {
		if om, ok := args[0].(px.OrderedMap); ok {
			t.InitFromHash(c, om)
			return
		}
	}
	panic(px.Error(px.Failure, issue.H{`message`: `internal error when creating an Object data type`}))
}

func (t *objectType) Accept(v px.Visitor, g px.Guard) {
	if g == nil {
		g = make(px.Guard)
	}
	if g.Seen(t, nil) {
		return
	}
	v(t)
	visitAnnotations(t.annotations, v, g)
	if t.parent != nil {
		t.parent.Accept(v, g)
	}
	t.parameters.EachValue(func(p interface{}) { p.(px.AnnotatedMember).Accept(v, g) })
	t.attributes.EachValue(func(a interface{}) { a.(px.AnnotatedMember).Accept(v, g) })
	t.functions.EachValue(func(f interface{}) { f.(px.AnnotatedMember).Accept(v, g) })
}

func (t *objectType) AttributesInfo() px.AttributesInfo {
	return t.attrInfo
}

func (t *objectType) Constructor(c px.Context) px.Function {
	if t.ctor == nil {
		t.createNewFunction(c)
	}
	return t.ctor
}

func (t *objectType) Default() px.Type {
	return objectTypeDefault
}

func (t *objectType) EachAttribute(includeParent bool, consumer func(attr px.Attribute)) {
	if includeParent && t.parent != nil {
		t.resolvedParent().EachAttribute(includeParent, consumer)
	}
	t.attributes.EachValue(func(a interface{}) { consumer(a.(px.Attribute)) })
}

func (t *objectType) EqualityAttributes() *hash.StringHash {
	eqa := make([]string, 0, 8)
	tp := t
	for tp != nil {
		if tp.equality != nil {
			eqa = append(eqa, tp.equality...)
		}
		tp = tp.resolvedParent()
	}
	attrs := hash.NewStringHash(len(eqa))
	for _, an := range eqa {
		attrs.Put(an, t.GetAttribute(an))
	}
	return attrs
}

func (t *objectType) Equals(other interface{}, guard px.Guard) bool {
	if t == other {
		return true
	}
	ot, ok := other.(*objectType)
	if !ok {
		return false
	}
	if t.initHashExpression != nil || ot.initHashExpression != nil {
		// Not yet resolved.
		return false
	}
	if t.name != ot.name {
		return false
	}
	if t.equalityIncludeType != ot.equalityIncludeType {
		return false
	}

	pa := t.resolvedParent()
	pb := ot.resolvedParent()
	if pa == nil {
		if pb != nil {
			return false
		}
	} else {
		if pb == nil {
			return false
		}
		if !pa.Equals(pb, guard) {
			return false
		}
	}
	if guard == nil {
		guard = make(px.Guard)
	}
	if guard.Seen(t, ot) {
		return true
	}
	return t.attributes.Equals(ot.attributes, guard) &&
		t.functions.Equals(ot.functions, guard) &&
		t.parameters.Equals(ot.parameters, guard) &&
		px.Equals(t.equality, ot.equality, guard) &&
		px.Equals(t.serialization, ot.serialization, guard)
}

func (t *objectType) FromReflectedValue(c px.Context, src reflect.Value) px.PuppetObject {
	if t.goType != nil {
		return NewReflectedValue(t, src).(px.PuppetObject)
	}
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	entries := t.appendAttributeValues(c, make([]*HashEntry, 0), &src)
	return px.New(c, t, WrapHash(entries)).(px.PuppetObject)
}

func (t *objectType) Get(key string) (value px.Value, ok bool) {
	if key == `_pcore_init_hash` {
		return t.InitHash(), true
	}
	return nil, false
}

func (t *objectType) GetAttribute(name string) px.Attribute {
	a, _ := t.attributes.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetAttribute(name)
		}
		return nil
	}).(px.Attribute)
	return a
}

func (t *objectType) GetFunction(name string) px.Function {
	f, _ := t.functions.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetFunction(name)
		}
		return nil
	}).(px.Function)
	return f
}

func (t *objectType) GetValue(key string, o px.Value) (value px.Value, ok bool) {
	if pu, ok := o.(px.ReadableObject); ok {
		return pu.Get(key)
	}
	return nil, false
}

func (t *objectType) GoType() reflect.Type {
	if t.goType != nil {
		return t.goType.Type()
	}
	return nil
}

func (t *objectType) HasHashConstructor() bool {
	return t.creators == nil || len(t.creators) == 2
}

func (t *objectType) parseAttributeType(c px.Context, receiverType, receiver string, typeString px.StringValue) px.Type {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				var label string
				if receiverType == `` {
					label = fmt.Sprintf(`%s.%s`, t.Label(), receiver)
				} else {
					label = fmt.Sprintf(`%s %s[%s]`, receiverType, t.Label(), receiver)
				}
				panic(px.Error(px.BadTypeString,
					issue.H{
						`string`: typeString,
						`label`:  label,
						`detail`: err.Error()}))
			}
			panic(r)
		}
	}()
	return c.ParseType(typeString.String())
}

func (t *objectType) InitFromHash(c px.Context, initHash px.OrderedMap) {
	px.AssertInstance(`object initializer`, TypeObjectInitHash, initHash)
	t.parameters = hash.EmptyStringHash
	t.attributes = hash.EmptyStringHash
	t.functions = hash.EmptyStringHash
	t.name = stringArg(initHash, keyName, t.name)

	if t.parent == nil {
		if pt, ok := initHash.Get4(keyParent); ok {
			switch pt := pt.(type) {
			case stringValue:
				t.parent = t.parseAttributeType(c, ``, `parent`, pt)
			case px.ResolvableType:
				t.parent = pt.Resolve(c)
			default:
				t.parent = pt.(px.Type)
			}
		}
	}

	parentMembers := hash.EmptyStringHash
	parentTypeParams := hash.EmptyStringHash
	var parentObjectType *objectType

	if t.parent != nil {
		t.checkSelfRecursion(c, t)
		parentObjectType = t.resolvedParent()
		parentMembers = parentObjectType.members(true)
		parentTypeParams = parentObjectType.typeParameters(true)
	}

	typeParameters := hashArg(initHash, keyTypeParameters)
	if !typeParameters.IsEmpty() {
		parameters := hash.NewStringHash(typeParameters.Len())
		typeParameters.EachPair(func(k, v px.Value) {
			key := k.String()
			var paramType px.Type
			var paramValue px.Value
			if ph, ok := v.(*Hash); ok {
				px.AssertInstance(
					func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
					TypeTypeParameter, ph)
				paramType = typeArg(ph, keyType, DefaultTypeType())
				paramValue = ph.Get5(keyValue, nil)
			} else {
				if tn, ok := v.(stringValue); ok {
					// Type name. Load the type.
					paramType = t.parseAttributeType(c, `type_parameter`, key, tn)
				} else {
					paramType = px.AssertInstance(
						func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
						DefaultTypeType(), v).(px.Type)
				}
				paramValue = nil
			}
			if _, ok := paramType.(*OptionalType); !ok {
				paramType = NewOptionalType(paramType)
			}
			param := newTypeParameter(c, key, t, WrapStringToInterfaceMap(c, issue.H{
				keyType:  paramType,
				keyValue: paramValue}))
			assertOverride(param, parentTypeParams)
			parameters.Put(key, param)
		})
		parameters.Freeze()
		t.parameters = parameters
	}

	constants := hashArg(initHash, keyConstants)
	attributes := hashArg(initHash, keyAttributes)
	attrSpecs := hash.NewStringHash(constants.Len() + attributes.Len())
	attributes.EachPair(func(k, v px.Value) {
		attrSpecs.Put(k.String(), v)
	})

	if !constants.IsEmpty() {
		constants.EachPair(func(k, v px.Value) {
			key := k.String()
			if attrSpecs.Includes(key) {
				panic(px.Error(px.BothConstantAndAttribute, issue.H{`label`: t.Label(), `key`: key}))
			}
			value := v.(px.Value)
			attrSpec := issue.H{
				keyType:  px.Generalize(value.PType()),
				keyValue: value,
				keyKind:  constant}
			attrSpec[keyOverride] = parentMembers.Includes(key)
			attrSpecs.Put(key, WrapStringToInterfaceMap(c, attrSpec))
		})
	}

	if !attrSpecs.IsEmpty() {
		ah := hash.NewStringHash(attrSpecs.Len())
		attrSpecs.EachPair(func(key string, ifv interface{}) {
			value := ifv.(px.Value)
			attrSpec, ok := value.(*Hash)
			if !ok {
				var attrType px.Type
				if tn, ok := value.(stringValue); ok {
					// Type name. Load the type.
					attrType = t.parseAttributeType(c, `attribute`, key, tn)
				} else {
					attrType = px.AssertInstance(
						func() string { return fmt.Sprintf(`attribute %s[%s]`, t.Label(), key) },
						DefaultTypeType(), value).(px.Type)
				}
				h := issue.H{keyType: attrType}
				if _, ok = attrType.(*OptionalType); ok {
					h[keyValue] = px.Undef
				}
				attrSpec = WrapStringToInterfaceMap(c, h)
			}
			attr := newAttribute(c, key, t, attrSpec)
			assertOverride(attr, parentMembers)
			ah.Put(key, attr)
		})
		ah.Freeze()
		t.attributes = ah
	}
	isInterface := t.attributes.IsEmpty() && (parentObjectType == nil || parentObjectType.isInterface)

	if t.goType != nil && t.attributes.IsEmpty() {
		if pt, ok := PrimitivePType(t.goType.Type()); ok {
			t.isInterface = false

			// Create the special attribute that holds the primitive value that is
			// reflectable to/from the the go type
			attrs := make([]*HashEntry, 2)
			attrs[0] = WrapHashEntry2(keyType, pt)
			attrs[1] = WrapHashEntry2(KeyGoName, stringValue(keyValue))
			ah := hash.NewStringHash(1)
			ah.Put(keyValue, newAttribute(c, keyValue, t, WrapHash(attrs)))
			ah.Freeze()
			t.attributes = ah
		}
	}

	funcSpecs := hashArg(initHash, keyFunctions)
	if funcSpecs.IsEmpty() {
		if isInterface && parentObjectType == nil {
			isInterface = false
		}
	} else {
		functions := hash.NewStringHash(funcSpecs.Len())
		funcSpecs.EachPair(func(key, value px.Value) {
			if attributes.IncludesKey(key) {
				panic(px.Error(px.MemberNameConflict, issue.H{`label`: fmt.Sprintf(`function %s[%s]`, t.Label(), key)}))
			}
			funcSpec, ok := value.(*Hash)
			if !ok {
				var funcType px.Type
				if tn, ok := value.(stringValue); ok {
					// Type name. Load the type.
					funcType = t.parseAttributeType(c, `function`, key.String(), tn)
				} else {
					funcType = px.AssertInstance(
						func() string { return fmt.Sprintf(`function %s[%s]`, t.Label(), key) },
						typeFunctionType, value).(px.Type)
				}
				funcSpec = WrapStringToInterfaceMap(c, issue.H{keyType: funcType})
			}
			fnc := newFunction(c, key.String(), t, funcSpec)
			assertOverride(fnc, parentMembers)
			functions.Put(key.String(), fnc)
		})
		functions.Freeze()
		t.functions = functions
	}
	t.equalityIncludeType = boolArg(initHash, keyEqualityIncludeType, true)

	var equality []string
	eq := initHash.Get5(keyEquality, nil)
	if es, ok := eq.(stringValue); ok {
		equality = []string{string(es)}
	} else if ea, ok := eq.(*Array); ok {
		equality = make([]string, ea.Len())
	} else {
		equality = nil
	}
	for _, attrName := range equality {
		mbr := t.attributes.Get2(attrName, func() interface{} {
			return t.functions.Get2(attrName, func() interface{} {
				return parentMembers.Get(attrName, nil)
			})
		})
		attr, ok := mbr.(px.Attribute)

		if !ok {
			if mbr == nil {
				panic(px.Error(px.EqualityAttributeNotFound, issue.H{`label`: t.Label(), `attribute`: attrName}))
			}
			panic(px.Error(px.EqualityNotAttribute, issue.H{`label`: t.Label(), `member`: mbr.(px.AnnotatedMember).Label()}))
		}
		if attr.Kind() == constant {
			panic(px.Error(px.EqualityOnConstant, issue.H{`label`: t.Label(), `attribute`: mbr.(px.AnnotatedMember).Label()}))
		}
		// Assert that attribute is not already include by parent equality
		if ok && parentObjectType != nil && parentObjectType.EqualityAttributes().Includes(attrName) {
			includingParent := t.findEqualityDefiner(attrName)
			panic(px.Error(px.EqualityRedefined, issue.H{`label`: t.Label(), `attribute`: attr.Label(), `including_parent`: includingParent}))
		}
	}
	t.equality = equality

	se, ok := initHash.Get5(keySerialization, nil).(*Array)
	if ok {
		serialization := make([]string, se.Len())
		var optFound px.Attribute
		se.EachWithIndex(func(elem px.Value, i int) {
			attrName := elem.String()
			mbr := t.attributes.Get2(attrName, func() interface{} {
				return t.functions.Get2(attrName, func() interface{} {
					return parentMembers.Get(attrName, nil)
				})
			})
			attr, ok := mbr.(px.Attribute)

			if !ok {
				if mbr == nil {
					panic(px.Error(px.SerializationAttributeNotFound, issue.H{`label`: t.Label(), `attribute`: attrName}))
				}
				panic(px.Error(px.SerializationNotAttribute, issue.H{`label`: t.Label(), `member`: mbr.(px.AnnotatedMember).Label()}))
			}
			if attr.Kind() == constant || attr.Kind() == derived {
				panic(px.Error(px.SerializationBadKind, issue.H{`label`: t.Label(), `kind`: attr.Kind(), `attribute`: attr.Label()}))
			}
			if attr.Kind() == givenOrDerived || attr.HasValue() {
				optFound = attr
			} else if optFound != nil {
				panic(px.Error(px.SerializationRequiredAfterOptional, issue.H{`label`: t.Label(), `required`: attr.Label(), `optional`: optFound.Label()}))
			}
			serialization[i] = attrName
		})
		t.serialization = serialization
	}

	t.isInterface = isInterface
	t.attrInfo = t.createAttributesInfo()
	t.annotatable.initialize(initHash.(*Hash))
	t.loader = c.Loader()
}

func (t *objectType) Implements(ifd px.ObjectType, g px.Guard) bool {
	if !ifd.IsInterface() {
		return false
	}

	for _, f := range ifd.Functions(true) {
		m, ok := t.Member(f.Name())
		if !ok {
			return false
		}
		mf, ok := m.(px.ObjFunc)
		if !ok {
			return false
		}
		if !f.Type().Equals(mf.Type(), g) {
			return false
		}
	}
	return true
}

func (t *objectType) InitHash() px.OrderedMap {
	return WrapStringPValue(t.initHash(true))
}

func (t *objectType) IsInterface() bool {
	return t.isInterface
}

func (t *objectType) IsAssignable(o px.Type, g px.Guard) bool {
	var ot *objectType
	switch o := o.(type) {
	case *objectType:
		ot = o
	case *objectTypeExtension:
		ot = o.baseType
	default:
		return false
	}

	if t == DefaultObjectType() {
		return true
	}

	if t.isInterface {
		return ot.Implements(t, g)
	}

	if t == DefaultObjectType() || t.Equals(ot, g) {
		return true
	}
	if ot.parent != nil {
		return t.IsAssignable(ot.parent, g)
	}
	return false
}

func (t *objectType) IsInstance(o px.Value, g px.Guard) bool {
	if po, ok := o.(px.Type); ok {
		return isAssignable(t, po.MetaType())
	}
	return isAssignable(t, o.PType())
}

func (t *objectType) IsParameterized() bool {
	if !t.parameters.IsEmpty() {
		return true
	}
	p := t.resolvedParent()
	if p != nil {
		return p.IsParameterized()
	}
	return false
}

func (t *objectType) IsMetaType() bool {
	return px.IsAssignable(AnyMetaType, t)
}

func (t *objectType) Label() string {
	if t.name == `` {
		return `Object`
	}
	return t.name
}

func (t *objectType) Member(name string) (px.CallableMember, bool) {
	mbr := t.attributes.Get2(name, func() interface{} {
		return t.functions.Get2(name, func() interface{} {
			if t.parent == nil {
				return nil
			}
			pm, _ := t.resolvedParent().Member(name)
			return pm
		})
	})
	if mbr == nil {
		return nil, false
	}
	return mbr.(px.CallableMember), true
}

func (t *objectType) MetaType() px.ObjectType {
	return ObjectMetaType
}

func (t *objectType) Name() string {
	return t.name
}

func (t *objectType) Parameters() []px.Value {
	return t.Parameters2(true)
}

func (t *objectType) Parameters2(includeName bool) []px.Value {
	if t == objectTypeDefault {
		return px.EmptyValues
	}
	return []px.Value{WrapStringPValue(t.initHash(includeName))}
}

func (t *objectType) Parent() px.Type {
	return t.parent
}

func (t *objectType) Resolve(c px.Context) px.Type {
	if t.initHashExpression == nil {
		return t
	}

	ihe := t.initHashExpression
	t.initHashExpression = nil

	if prt, ok := t.parent.(px.ResolvableType); ok {
		t.parent = resolveTypeRefs(c, prt).(px.Type)
	}

	var initHash px.OrderedMap
	switch ihe := ihe.(type) {
	case px.OrderedMap:
		initHash = resolveTypeRefs(c, ihe).(px.OrderedMap)
	case *taggedType:
		t.goType = ihe
		initHash = c.Reflector().InitializerFromTagged(t.name, t.parent, ihe)
		c.ImplementationRegistry().RegisterType(t, ihe.Type())
	default:
		tg := px.NewTaggedType(reflect.TypeOf(ihe), nil)
		t.goType = tg
		initHash = c.Reflector().InitializerFromTagged(t.name, t.parent, tg)
		c.ImplementationRegistry().RegisterType(t, tg.Type())
	}
	t.InitFromHash(c, initHash)
	return t
}

func (t *objectType) CanSerializeAsString() bool {
	return t == objectTypeDefault
}

func (t *objectType) SerializationString() string {
	return t.String()
}

func (t *objectType) String() string {
	return px.ToString(t)
}

func (t *objectType) ToKey() px.HashKey {
	return t.hashKey
}

func (t *objectType) ToReflectedValue(c px.Context, src px.PuppetObject, dest reflect.Value) {
	dt := dest.Type()
	rf := c.Reflector()
	fs := rf.Fields(dt)
	for i, field := range fs {
		f := dest.Field(i)
		if field.Anonymous && i == 0 && t.parent != nil {
			t.resolvedParent().ToReflectedValue(c, src, f)
			continue
		}
		an := rf.FieldName(&field)
		if av, ok := src.Get(an); ok {
			rf.ReflectTo(av, f)
			continue
		}
		panic(px.Error(px.AttributeNotFound, issue.H{`name`: an}))
	}
}

func (t *objectType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), t.PType())
	switch f.FormatChar() {
	case 's', 'p':
		quoted := f.IsAlt() && f.FormatChar() == 's'
		if quoted || f.HasStringFlags() {
			bld := bytes.NewBufferString(``)
			t.basicTypeToString(bld, f, s, g)
			f.ApplyStringFlags(b, bld.String(), quoted)
		} else {
			t.basicTypeToString(b, f, s, g)
		}
	default:
		panic(s.UnsupportedFormat(t.PType(), `sp`, f))
	}
}

func (t *objectType) basicTypeToString(b io.Writer, f px.Format, s px.FormatContext, g px.RDetect) {

	if t.Equals(DefaultObjectType(), nil) {
		utils.WriteString(b, `Object`)
		return
	}

	typeSetParent := false
	typeSetName, inTypeSet := s.Property(`typeSet`)
	if tp, ok := s.Property(`typeSetParent`); ok && tp == `true` {
		s = s.WithProperties(map[string]string{`typeSetParent`: `false`})
		typeSetParent = true
	}

	if ex, ok := s.Property(`expanded`); !(ok && ex == `true`) {
		name := t.Name()
		if name != `` {
			if inTypeSet {
				name = stripTypeSetName(typeSetName, name)
			}
			utils.WriteString(b, name)
			return
		}
	}

	// Avoid nested expansions
	s = s.WithProperties(map[string]string{`expanded`: `false`})

	indent1 := s.Indentation()
	indent2 := indent1.Increase(f.IsAlt())
	indent3 := indent2.Increase(f.IsAlt())
	padding1 := ``
	padding2 := ``
	padding3 := ``
	if f.IsAlt() {
		padding1 = indent1.Padding()
		padding2 = indent2.Padding()
		padding3 = indent3.Padding()
	}

	cf := f.ContainerFormats()
	if cf == nil {
		cf = DefaultContainerFormats
	}

	ctx2 := px.NewFormatContext2(indent2, s.FormatMap(), s.Properties())
	cti2 := px.NewFormatContext2(indent2, cf, s.Properties())
	ctx3 := px.NewFormatContext2(indent3, s.FormatMap(), s.Properties())

	if inTypeSet {
		if t.parent != nil {
			utils.WriteString(b, stripTypeSetName(typeSetName, t.parent.Name()))
		}
	}
	if !typeSetParent {
		utils.WriteString(b, `Object[`)
	}
	utils.WriteString(b, `{`)

	first2 := true
	ih := t.initHash(!inTypeSet)
	for _, key := range ih.Keys() {
		if inTypeSet && key == `parent` {
			continue
		}

		value := ih.Get(key, nil).(px.Value)
		if first2 {
			first2 = false
		} else {
			utils.WriteString(b, `,`)
			if !f.IsAlt() {
				utils.WriteString(b, ` `)
			}
		}
		if f.IsAlt() {
			utils.WriteString(b, "\n")
			utils.WriteString(b, padding2)
		}
		utils.WriteString(b, key)
		utils.WriteString(b, ` => `)
		switch key {
		case `attributes`, `functions`:
			// The keys should not be quoted in this hash
			utils.WriteString(b, `{`)
			first3 := true
			value.(*Hash).EachPair(func(name, typ px.Value) {
				if first3 {
					first3 = false
				} else {
					utils.WriteString(b, `,`)
					if !f.IsAlt() {
						utils.WriteString(b, ` `)
					}
				}
				if f.IsAlt() {
					utils.WriteString(b, "\n")
					utils.WriteString(b, padding3)
				}
				utils.PuppetQuote(b, name.String())
				utils.WriteString(b, ` => `)
				typ.ToString(b, ctx3, g)
			})
			if f.IsAlt() {
				utils.WriteString(b, "\n")
				utils.WriteString(b, padding2)
			}
			utils.WriteString(b, "}")
		default:
			cx := cti2
			if isContainer(value, s) {
				cx = ctx2
			}
			value.ToString(b, cx, g)
		}
	}
	if f.IsAlt() {
		utils.WriteString(b, "\n")
		utils.WriteString(b, padding1)
	}
	utils.WriteString(b, "}")
	if !typeSetParent {
		utils.WriteString(b, "]")
	}
}

func (t *objectType) PType() px.Type {
	return &TypeType{t}
}

func (t *objectType) appendAttributeValues(c px.Context, entries []*HashEntry, src *reflect.Value) []*HashEntry {
	dt := src.Type()
	fs := Fields(dt)

	for i, field := range fs {
		sf := src.Field(i)
		if sf.Kind() == reflect.Ptr {
			sf = sf.Elem()
		}
		if field.Anonymous && i == 0 && t.parent != nil {
			entries = t.resolvedParent().appendAttributeValues(c, entries, &sf)
		} else {
			if sf.IsValid() {
				switch sf.Kind() {
				case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
					if sf.IsNil() {
						continue
					}
				}
				entries = append(entries, WrapHashEntry2(FieldName(&field), wrap(c, sf)))
			}
		}
	}
	return entries
}

func (t *objectType) checkSelfRecursion(c px.Context, originator *objectType) {
	if t.parent != nil {
		op := t.resolvedParent()
		if op.Equals(originator, nil) {
			panic(px.Error(px.ObjectInheritsSelf, issue.H{`label`: originator.Label()}))
		}
		op.checkSelfRecursion(c, originator)
	}
}

func (t *objectType) collectAttributes(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectAttributes(true, collector)
	}
	collector.PutAll(t.attributes)
}

func (t *objectType) Functions(includeParent bool) []px.ObjFunc {
	collector := hash.NewStringHash(7)
	t.collectFunctions(includeParent, collector)
	vs := collector.Values()
	fs := make([]px.ObjFunc, len(vs))
	for i, v := range vs {
		fs[i] = v.(px.ObjFunc)
	}
	return fs
}

func (t *objectType) collectFunctions(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectFunctions(true, collector)
	}
	collector.PutAll(t.functions)
}

func (t *objectType) collectMembers(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectMembers(true, collector)
	}
	collector.PutAll(t.attributes)
	collector.PutAll(t.functions)
}

func (t *objectType) collectParameters(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectParameters(true, collector)
	}
	collector.PutAll(t.parameters)
}

func (t *objectType) createAttributesInfo() *attributesInfo {
	attrs := make([]px.Attribute, 0)
	nonOptSize := 0
	if t.serialization == nil {
		optAttrs := make([]px.Attribute, 0)
		t.EachAttribute(true, func(attr px.Attribute) {
			switch attr.Kind() {
			case constant, derived:
			case givenOrDerived:
				optAttrs = append(optAttrs, attr)
			default:
				if attr.HasValue() {
					optAttrs = append(optAttrs, attr)
				} else {
					attrs = append(attrs, attr)
				}
			}
		})
		nonOptSize = len(attrs)
		attrs = append(attrs, optAttrs...)
	} else {
		atMap := hash.NewStringHash(15)
		t.collectAttributes(true, atMap)
		for _, key := range t.serialization {
			attr := atMap.Get(key, nil).(px.Attribute)
			if attr.HasValue() {
				nonOptSize++
			}
			attrs = append(attrs, attr)
		}
	}
	return newAttributesInfo(attrs, nonOptSize, t.EqualityAttributes().Keys())
}

func (t *objectType) createInitType() *StructType {
	if t.initType == nil {
		ai := t.attrInfo
		if ai == nil {
			// Default Object. It has no attributes
			t.initType = NewStructType([]*StructElement{})
			return t.initType
		}
		attrs := ai.attributes
		elements := make([]*StructElement, len(attrs))
		t.initType = NewStructType(elements)
		for i, attr := range attrs {
			at := typeAndInit(attr.Type())
			switch attr.Kind() {
			case constant, derived:
			case givenOrDerived:
				elements[i] = NewStructElement(newOptionalType3(attr.Name()), at)
			default:
				var key px.Type
				if attr.HasValue() {
					key = newOptionalType3(attr.Name())
				} else {
					key = NewStringType(nil, attr.Name())
				}
				elements[i] = NewStructElement(key, at)
			}
		}
	}
	return t.initType
}

func (t *objectType) createNewFunction(c px.Context) {
	pi := t.AttributesInfo()

	var functions []px.DispatchFunction
	if t.creators != nil {
		functions = t.creators
		if functions[0] == nil {
			// Specific instruction not to create a constructor
			return
		}
	} else {
		if t.name != `` {
			var dl px.DefiningLoader
			if t.loader == nil {
				dl = c.DefiningLoader()
			} else {
				dl = t.loader.(px.DefiningLoader)
			}
			tn := px.NewTypedName(px.NsAllocator, t.name)
			le := dl.LoadEntry(c, tn)
			if le == nil || le.Value() == nil {
				dl.SetEntry(tn, px.NewLoaderEntry(px.MakeGoAllocator(func(ctx px.Context, args []px.Value) px.Value {
					return AllocObjectValue(t)
				}), nil))
			}
		}

		functions = []px.DispatchFunction{
			// Positional argument creator
			func(c px.Context, args []px.Value) px.Value {
				return NewObjectValue(c, t, args)
			},
			// Named argument creator
			func(c px.Context, args []px.Value) px.Value {
				ai := t.AttributesInfo()
				if oh, ok := args[0].(*Hash); ok {
					el := make([]*HashEntry, 0, oh.Len())
					for _, ca := range ai.Attributes() {
						if e, ok := oh.GetEntry(ca.Name()); ok {
							el = append(el, WrapHashEntry(e.Key(), coerceTo(c, []string{ca.Label()}, ca.Type(), e.Value())))
						}
					}
					return newObjectValue2(c, t, oh.Merge(WrapHash(el)).(*Hash))
				}
				panic(px.MismatchError(t.Label(), t, args[0]))
			}}
	}

	paCreator := func(d px.Dispatch) {
		for i, attr := range pi.Attributes() {
			at := attr.Type()
			if ot, ok := at.(px.ObjectType); ok {
				at = typeAndInit(ot)
			}
			switch attr.Kind() {
			case constant, derived:
			case givenOrDerived:
				d.OptionalParam2(at)
			default:
				if i >= pi.RequiredCount() {
					d.OptionalParam2(at)
				} else {
					d.Param2(at)
				}
			}
		}
		d.Function(functions[0])
	}

	var creators []px.DispatchCreator
	if len(functions) > 1 {
		// A named argument constructor exists. Place it first.
		creators = []px.DispatchCreator{func(d px.Dispatch) {
			d.Param2(t.createInitType())
			d.Function(functions[1])
		}, paCreator}
	} else {
		creators = []px.DispatchCreator{paCreator}
	}

	t.ctor = px.MakeGoConstructor(t.name, creators...).Resolve(c)
}

func typeAndInit(t px.Type) px.Type {
	switch t := t.(type) {
	case *objectType:
		return NewVariantType(t, t.createInitType())
	case *HashType:
		return NewHashType(typeAndInit(t.keyType), typeAndInit(t.valueType), t.size)
	case *ArrayType:
		return NewArrayType(typeAndInit(t.ElementType()), t.size)
	case *StructType:
		ses := make([]*StructElement, len(t.elements))
		for i, se := range t.elements {
			ses[i] = &StructElement{key: se.key, name: se.name, value: typeAndInit(se.value)}
		}
		return NewStructType(ses)
	case *TupleType:
		tts := make([]px.Type, len(t.types))
		for i, tt := range t.types {
			tts[i] = typeAndInit(tt)
		}
		return &TupleType{types: tts, size: t.size, givenOrActualSize: t.givenOrActualSize}
	case *VariantType:
		tts := make([]px.Type, len(t.types))
		for i, tt := range t.types {
			tts[i] = typeAndInit(tt)
		}
		return &VariantType{types: tts}
	case *OptionalType:
		return &OptionalType{typ: typeAndInit(t.typ)}
	case *NotUndefType:
		return &OptionalType{typ: typeAndInit(t.typ)}
	default:
		return t
	}
}

func (t *objectType) findEqualityDefiner(attrName string) *objectType {
	tp := t
	for tp != nil {
		p := tp.resolvedParent()
		if p == nil || !p.EqualityAttributes().Includes(attrName) {
			return tp
		}
		tp = p
	}
	return nil
}

func (t *objectType) initHash(includeName bool) *hash.StringHash {
	h := t.annotatable.initHash()
	if includeName && t.name != `` && t.name != `Object` {
		h.Put(keyName, stringValue(t.name))
	}
	if t.parent != nil {
		h.Put(keyParent, t.parent)
	}
	if !t.parameters.IsEmpty() {
		h.Put(keyTypeParameters, compressedMembersHash(t.parameters))
	}
	if !t.attributes.IsEmpty() {
		// Divide attributes into constants and others
		constants := make([]*HashEntry, 0)
		others := hash.NewStringHash(5)
		t.attributes.EachPair(func(key string, value interface{}) {
			a := value.(px.Attribute)
			if a.Kind() == constant && px.Equals(a.Type(), px.Generalize(a.Value().PType()), nil) {
				constants = append(constants, WrapHashEntry2(key, a.Value()))
			} else {
				others.Put(key, a)
			}
			if !others.IsEmpty() {
				h.Put(keyAttributes, compressedMembersHash(others))
			}
			if len(constants) > 0 {
				h.Put(keyConstants, WrapHash(constants))
			}
		})
	}
	if !t.functions.IsEmpty() {
		h.Put(keyFunctions, compressedMembersHash(t.functions))
	}
	if t.equality != nil {
		ev := make([]px.Value, len(t.equality))
		for i, e := range t.equality {
			ev[i] = stringValue(e)
		}
		h.Put(keyEquality, WrapValues(ev))
	}

	if !t.equalityIncludeType {
		h.Put(keyEqualityIncludeType, BooleanFalse)
	}

	if t.serialization != nil {
		sv := make([]px.Value, len(t.serialization))
		for i, s := range t.serialization {
			sv[i] = stringValue(s)
		}
		h.Put(keySerialization, WrapValues(sv))
	}
	return h
}

func (t *objectType) members(includeParent bool) *hash.StringHash {
	collector := hash.NewStringHash(7)
	t.collectMembers(includeParent, collector)
	return collector
}

func (t *objectType) resolvedParent() *objectType {
	tp := t.parent
	for {
		switch at := tp.(type) {
		case nil:
			return nil
		case *objectType:
			return at
		case *TypeAliasType:
			tp = at.resolvedType
		default:
			panic(px.Error(px.IllegalObjectInheritance, issue.H{`label`: t.Label(), `type`: tp.PType().String()}))
		}
	}
}

// setCreators takes one or two arguments. The first function is for positional arguments, the second
// for named arguments (expects exactly one argument which is a Hash.
func (t *objectType) setCreators(creators ...px.DispatchFunction) {
	t.creators = creators
}

func (t *objectType) typeParameters(includeParent bool) *hash.StringHash {
	c := hash.NewStringHash(5)
	t.collectParameters(includeParent, c)
	return c
}

func compressedMembersHash(mh *hash.StringHash) *Hash {
	he := make([]*HashEntry, 0, mh.Len())
	mh.EachPair(func(key string, value interface{}) {
		fh := value.(px.AnnotatedMember).InitHash()
		if fh.Len() == 1 {
			tp := fh.Get5(keyType, nil)
			if tp != nil {
				he = append(he, WrapHashEntry2(key, tp))
				return
			}
		}
		he = append(he, WrapHashEntry2(key, fh))
	})
	return WrapHash(he)
}

func resolveTypeRefs(c px.Context, v interface{}) px.Value {
	switch v := v.(type) {
	case *Hash:
		he := make([]*HashEntry, v.Len())
		i := 0
		v.EachPair(func(key, value px.Value) {
			he[i] = WrapHashEntry(
				resolveTypeRefs(c, key), resolveTypeRefs(c, value))
			i++
		})
		return WrapHash(he)
	case *Array:
		ae := make([]px.Value, v.Len())
		i := 0
		v.Each(func(value px.Value) {
			ae[i] = resolveTypeRefs(c, value)
			i++
		})
		return WrapValues(ae)
	case px.ResolvableType:
		return v.Resolve(c)
	default:
		return v.(px.Value)
	}
}

func newObjectType(name, typeDecl string, creators ...px.DispatchFunction) px.ObjectType {
	ta, err := Parse(typeDecl)
	if err != nil {
		_, fileName, fileLine, _ := runtime.Caller(1)
		panic(convertReported(err, fileName, fileLine))
	}
	if h, ok := ta.(*Hash); ok {
		// "type = {}"
		return MakeObjectType(name, nil, h, true, creators...)
	}
	if dt, ok := ta.(*DeferredType); ok {
		ps := dt.Parameters()
		if len(ps) == 1 {
			if h, ok := ps[0].(*Hash); ok {
				var p px.Type
				if pn := dt.Name(); pn != `TypeSet` && pn != `Object` {
					p = NewTypeReferenceType(pn)
				}
				return MakeObjectType(name, p, h, true, creators...)
			}
		}
	}
	panic(px.Error(px.NoDefinition, issue.H{`source`: ``, `type`: px.NsType, `name`: name}))
}

func newObjectType2(c px.Context, args ...px.Value) *objectType {
	argc := len(args)
	switch argc {
	case 0:
		return DefaultObjectType()
	case 1:
		arg := args[0]
		if initHash, ok := arg.(*Hash); ok {
			if initHash.IsEmpty() {
				return DefaultObjectType()
			}
			obj := AllocObjectType()
			obj.InitFromHash(c, initHash)
			obj.loader = c.Loader()
			return obj
		}
		panic(illegalArgumentType(`Object[]`, 0, `Hash[String,Any]`, arg.PType()))
	default:
		panic(illegalArgumentCount(`Object[]`, `1`, argc))
	}
}

// MakeObjectType creates a new object type and optionally registers it as a resolvable to be picked up for resolution
// on the next call to px.ResolveResolvables if the given register flag is true. This flag should only be set to true
// when the call stems from a static init() function where no context is available.
func MakeObjectType(name string, parent px.Type, initHash px.OrderedMap, register bool, creators ...px.DispatchFunction) *objectType {
	if name == `` {
		name = initHash.Get5(`name`, emptyString).String()
	}
	obj := AllocObjectType()
	obj.name = name
	obj.initHashExpression = initHash
	obj.parent = parent
	obj.setCreators(creators...)
	if register {
		registerResolvableType(obj)
	}
	return obj
}

func newGoObjectType(name string, rType reflect.Type, typeDecl string, creators ...px.DispatchFunction) px.ObjectType {
	t := newObjectType(name, typeDecl, creators...)
	registerMapping(t, rType)
	return t
}
