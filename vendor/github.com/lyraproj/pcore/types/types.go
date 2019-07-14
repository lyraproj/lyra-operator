package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"sync"
	"time"

	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/semver/semver"
)

const (
	NoString = "\x00"

	HkBinary       = byte('B')
	HkBoolean      = byte('b')
	HkDefault      = byte('d')
	HkFloat        = byte('f')
	HkInteger      = byte('i')
	HkRegexp       = byte('r')
	HkTimespan     = byte('D')
	HkTimestamp    = byte('T')
	HkType         = byte('t')
	HkUndef        = byte('u')
	HkUri          = byte('U')
	HkVersion      = byte('v')
	HkVersionRange = byte('R')

	IntegerHex = `(?:0[xX][0-9A-Fa-f]+)`
	IntegerOct = `(?:0[0-7]+)`
	IntegerBin = `(?:0[bB][01]+)`
	IntegerDec = `(?:0|[1-9]\d*)`
	SignPrefix = `[+-]?\s*`

	OptionalFraction = `(?:\.\d+)?`
	OptionalExponent = `(?:[eE]-?\d+)?`
	FloatDec         = `(?:` + IntegerDec + OptionalFraction + OptionalExponent + `)`

	IntegerPattern = `\A` + SignPrefix + `(?:` + IntegerDec + `|` + IntegerHex + `|` + IntegerOct + `|` + IntegerBin + `)\z`
	FloatPattern   = `\A` + SignPrefix + `(?:` + FloatDec + `|` + IntegerHex + `|` + IntegerOct + `|` + IntegerBin + `)\z`
)

// isInstance answers if value is an instance of the given puppetType
func isInstance(puppetType px.Type, value px.Value) bool {
	return GuardedIsInstance(puppetType, value, nil)
}

// isAssignable answers if t is assignable to this type
func isAssignable(puppetType px.Type, other px.Type) bool {
	return GuardedIsAssignable(puppetType, other, nil)
}

func generalize(a px.Type) px.Type {
	if g, ok := a.(px.Generalizable); ok {
		return g.Generic()
	}
	if g, ok := a.(px.ParameterizedType); ok {
		return g.Default()
	}
	return a
}

func defaultFor(t px.Type) px.Type {
	if g, ok := t.(px.ParameterizedType); ok {
		return g.Default()
	}
	return t
}

func normalize(t px.Type) px.Type {
	// TODO: Implement for ParameterizedType
	return t
}

func resolve(c px.Context, t px.Type) px.Type {
	if rt, ok := t.(px.ResolvableType); ok {
		return rt.Resolve(c)
	}
	return t
}

func EachCoreType(fc func(t px.Type)) {
	keys := make([]string, len(coreTypes))
	i := 0
	for key := range coreTypes {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	for _, key := range keys {
		fc(coreTypes[key])
	}
}

func GuardedIsInstance(a px.Type, v px.Value, g px.Guard) bool {
	return a.IsInstance(v, g)
}

func GuardedIsAssignable(a px.Type, b px.Type, g px.Guard) bool {
	if a == b || a == anyTypeDefault {
		return true
	}
	switch b := b.(type) {
	case nil:
		return false
	case *UnitType:
		return true
	case *NotUndefType:
		nt := b.typ
		if !GuardedIsAssignable(nt, undefTypeDefault, g) {
			return GuardedIsAssignable(a, nt, g)
		}
	case *OptionalType:
		if GuardedIsAssignable(a, undefTypeDefault, g) {
			ot := b.typ
			return ot == nil || GuardedIsAssignable(a, ot, g)
		}
		return false
	case *TypeAliasType:
		return GuardedIsAssignable(a, b.resolvedType, g)
	case *VariantType:
		return b.allAssignableTo(a, g)
	}
	return a.IsAssignable(b, g)
}

func UniqueTypes(types []px.Type) []px.Type {
	top := len(types)
	if top < 2 {
		return types
	}

	result := make([]px.Type, 0, top)
	exists := make(map[px.HashKey]bool, top)
	for _, t := range types {
		key := px.ToKey(t)
		if !exists[key] {
			exists[key] = true
			result = append(result, t)
		}
	}
	return result
}

// ValueSlice convert a slice of values that implement the pcore.Value interface to []pcore.Value. The
// method will panic if the given argument is not a slice or array, or if not all
// elements implement the pcore.Value interface
func ValueSlice(slice interface{}) []px.Value {
	sv := reflect.ValueOf(slice)
	top := sv.Len()
	result := make([]px.Value, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = sv.Index(idx).Interface().(px.Value)
	}
	return result
}

func UniqueValues(values []px.Value) []px.Value {
	top := len(values)
	if top < 2 {
		return values
	}

	result := make([]px.Value, 0, top)
	exists := make(map[px.HashKey]bool, top)
	for _, v := range values {
		key := px.ToKey(v)
		if !exists[key] {
			exists[key] = true
			result = append(result, v)
		}
	}
	return result
}

func illegalArgument(name string, index int, arg string) issue.Reported {
	return issue.NewReported(px.IllegalArgument, issue.SeverityError, issue.H{`function`: name, `index`: index, `arg`: arg}, 1)
}

func illegalArguments(name string, message string) issue.Reported {
	return issue.NewReported(px.IllegalArguments, issue.SeverityError, issue.H{`function`: name, `message`: message}, 1)
}

func illegalArgumentType(name string, index int, expected string, actual px.Value) issue.Reported {
	return issue.NewReported(px.IllegalArgumentType, issue.SeverityError, issue.H{`function`: name, `index`: index, `expected`: expected, `actual`: px.DetailedValueType(actual).String()}, 1)
}

func illegalArgumentCount(name string, expected string, actual int) issue.Reported {
	return issue.NewReported(px.IllegalArgumentCount, issue.SeverityError, issue.H{`function`: name, `expected`: expected, `actual`: actual}, 1)
}

func TypeToString(t px.Type, b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), t.PType())
	switch f.FormatChar() {
	case 's', 'p':
		quoted := f.IsAlt() && f.FormatChar() == 's'
		if quoted || f.HasStringFlags() {
			bld := bytes.NewBufferString(``)
			basicTypeToString(t, bld, s, g)
			f.ApplyStringFlags(b, bld.String(), quoted)
		} else {
			basicTypeToString(t, b, s, g)
		}
	default:
		panic(s.UnsupportedFormat(t.PType(), `sp`, f))
	}
}

func basicTypeToString(t px.Type, b io.Writer, s px.FormatContext, g px.RDetect) {
	name := t.Name()
	if tp, ok := s.Property(`typeSetParent`); ok && tp == `true` {
		s = s.WithProperties(map[string]string{`typeSetParent`: `false`})
	}

	if ex, ok := s.Property(`expanded`); !(ok && ex == `true`) {
		switch t.(type) {
		case *TypeAliasType:
			if ts, ok := s.Property(`typeSet`); ok {
				name = stripTypeSetName(ts, name)
			}
			_, err := io.WriteString(b, name)
			if err != nil {
				panic(err)
			}
			return
		}
	}
	_, err := io.WriteString(b, name)
	if err != nil {
		panic(err)
	}
	if pt, ok := t.(px.ParameterizedType); ok {
		params := pt.Parameters()
		if len(params) > 0 {
			WrapValues(params).ToString(b, s.Subsequent(), g)
		}
	}
}

func stripTypeSetName(tsName, name string) string {
	tsName = tsName + `::`
	if strings.HasPrefix(name, tsName) {
		// Strip name and two colons
		return name[len(tsName):]
	}
	return name
}

type alterFunc func(t px.Type) px.Type

func alterTypes(types []px.Type, function alterFunc) []px.Type {
	al := make([]px.Type, len(types))
	for idx, t := range types {
		al[idx] = function(t)
	}
	return al
}

func toTypes(types px.List) ([]px.Type, int) {
	top := types.Len()
	if top == 1 {
		if a, ok := types.At(0).(px.List); ok {
			if _, ok = a.(stringValue); !ok {
				ts, f := toTypes(a)
				if f >= 0 {
					return nil, 0
				}
				return ts, 0
			}
		}
	}
	result := make([]px.Type, 0, top)
	if types.All(func(t px.Value) bool {
		if pt, ok := t.(px.Type); ok {
			result = append(result, pt)
			return true
		}
		return false
	}) {
		return result, -1
	}
	return nil, 0
}

func DefaultDataType() *TypeAliasType {
	return dataTypeDefault
}

func DefaultRichDataType() *TypeAliasType {
	return richDataTypeDefault
}

func NilAs(dflt, t px.Type) px.Type {
	if t == nil {
		t = dflt
	}
	return t
}

func CopyAppend(types []px.Type, t px.Type) []px.Type {
	top := len(types)
	tc := make([]px.Type, top+1)
	copy(tc, types)
	tc[top] = t
	return tc
}

var dataArrayTypeDefault = &ArrayType{IntegerTypePositive, &TypeReferenceType{`Data`}}
var dataHashTypeDefault = &HashType{IntegerTypePositive, stringTypeDefault, &TypeReferenceType{`Data`}}
var dataTypeDefault = &TypeAliasType{name: `Data`, resolvedType: &VariantType{[]px.Type{scalarDataTypeDefault, undefTypeDefault, dataArrayTypeDefault, dataHashTypeDefault}}}

var richKeyTypeDefault = &VariantType{[]px.Type{stringTypeDefault, numericTypeDefault}}
var richDataArrayTypeDefault = &ArrayType{IntegerTypePositive, &TypeReferenceType{`RichData`}}
var richDataHashTypeDefault = &HashType{IntegerTypePositive, richKeyTypeDefault, &TypeReferenceType{`RichData`}}
var richDataTypeDefault = &TypeAliasType{`RichData`, nil, &VariantType{
	[]px.Type{scalarTypeDefault,
		binaryTypeDefault,
		defaultTypeDefault,
		objectTypeDefault,
		typeTypeDefault,
		typeSetTypeDefault,
		undefTypeDefault,
		richDataArrayTypeDefault,
		richDataHashTypeDefault}}, nil}

type Mapping struct {
	T px.Type
	R reflect.Type
}

var resolvableTypes = make([]px.ResolvableType, 0, 16)
var resolvableMappings = make([]Mapping, 0, 16)
var resolvableTypesLock sync.Mutex

type BuildFunctionArgs struct {
	Name       string
	LocalTypes px.LocalTypesCreator
	Creators   []px.DispatchCreator
}

var constructorsDecls = make([]*BuildFunctionArgs, 0, 16)

func init() {
	// "resolve" the dataType and richDataType
	dataArrayTypeDefault.typ = dataTypeDefault
	dataHashTypeDefault.valueType = dataTypeDefault
	richDataArrayTypeDefault.typ = richDataTypeDefault
	richDataHashTypeDefault.valueType = richDataTypeDefault

	px.DefaultFor = defaultFor
	px.Generalize = generalize
	px.Normalize = normalize
	px.IsAssignable = isAssignable
	px.IsInstance = isInstance
	px.New = newInstance

	px.DetailedValueType = func(value px.Value) px.Type {
		if dt, ok := value.(px.DetailedTypeValue); ok {
			return dt.DetailedType()
		}
		return value.PType()
	}

	px.GenericType = func(t px.Type) px.Type {
		if g, ok := t.(px.Generalizable); ok {
			return g.Generic()
		}
		return t
	}

	px.GenericValueType = func(value px.Value) px.Type {
		return px.GenericType(value.PType())
	}

	px.ToArray = func(elements []px.Value) px.List {
		return WrapValues(elements)
	}

	px.ToKey = func(value px.Value) px.HashKey {
		if hk, ok := value.(px.HashKeyValue); ok {
			return hk.ToKey()
		}
		b := bytes.NewBuffer([]byte{})
		appendKey(b, value)
		return px.HashKey(b.String())
	}

	px.IsTruthy = func(tv px.Value) bool {
		switch tv := tv.(type) {
		case *UndefValue:
			return false
		case booleanValue:
			return tv.Bool()
		default:
			return true
		}
	}

	px.NewObjectType = newObjectType
	px.NewGoObjectType = newGoObjectType
	px.NewNamedType = newNamedType
	px.NewGoType = newGoType
	px.RegisterResolvableType = registerResolvableType
	px.NewGoConstructor = newGoConstructor
	px.NewGoConstructor2 = newGoConstructor2
	px.Wrap = wrap
	px.WrapReflected = wrapReflected
	px.WrapReflectedType = wrapReflectedType
}

func canSerializeAsString(t px.Type) bool {
	if t == nil {
		// true because nil members will not participate
		return true
	}
	if st, ok := t.(px.SerializeAsString); ok {
		return st.CanSerializeAsString()
	}
	return false
}

// New creates a new instance of type t
func newInstance(c px.Context, receiver px.Value, args ...px.Value) px.Value {
	var name string
	typ, ok := receiver.(px.Type)
	if ok {
		name = typ.Name()
	} else {
		// Type might be in string form
		_, ok = receiver.(stringValue)
		if !ok {
			// Only types or names of types can be used
			panic(px.Error(px.InstanceDoesNotRespond, issue.H{`type`: receiver.PType(), `message`: `new`}))
		}

		name = receiver.String()
		var t interface{}
		if t, ok = px.Load(c, NewTypedName(px.NsType, name)); ok {
			typ = t.(px.Type)
		}
	}

	if nb, ok := typ.(px.Newable); ok {
		return nb.New(c, args)
	}

	var ctor px.Function
	var ct px.Creatable
	ct, ok = typ.(px.Creatable)
	if ok {
		ctor = ct.Constructor(c)
	}

	if ctor == nil {
		tn := NewTypedName(px.NsConstructor, name)
		if t, ok := px.Load(c, tn); ok {
			ctor = t.(px.Function)
		}
	}

	if ctor == nil {
		panic(px.Error(px.InstanceDoesNotRespond, issue.H{`type`: name, `message`: `new`}))
	}

	r := ctor.(px.Function).Call(c, nil, args...)
	if typ != nil {
		px.AssertInstance(`new`, typ, r)
	}
	return r
}

func newGoConstructor(typeName string, creators ...px.DispatchCreator) {
	registerGoConstructor(&BuildFunctionArgs{typeName, nil, creators})
}

func newGoConstructor2(typeName string, localTypes px.LocalTypesCreator, creators ...px.DispatchCreator) {
	registerGoConstructor(&BuildFunctionArgs{typeName, localTypes, creators})
}

func newGoConstructor3(typeNames []string, localTypes px.LocalTypesCreator, creators ...px.DispatchCreator) {
	for _, tn := range typeNames {
		registerGoConstructor(&BuildFunctionArgs{tn, localTypes, creators})
	}
}

func PopDeclaredTypes() (types []px.ResolvableType) {
	resolvableTypesLock.Lock()
	types = resolvableTypes
	if len(types) > 0 {
		resolvableTypes = make([]px.ResolvableType, 0, 16)
	}
	resolvableTypesLock.Unlock()
	return
}

func PopDeclaredMappings() (types []Mapping) {
	resolvableTypesLock.Lock()
	types = resolvableMappings
	if len(types) > 0 {
		resolvableMappings = make([]Mapping, 0, 16)
	}
	resolvableTypesLock.Unlock()
	return
}

func PopDeclaredConstructors() (ctorDecls []*BuildFunctionArgs) {
	resolvableTypesLock.Lock()
	ctorDecls = constructorsDecls
	if len(ctorDecls) > 0 {
		constructorsDecls = make([]*BuildFunctionArgs, 0, 16)
	}
	resolvableTypesLock.Unlock()
	return
}

func registerGoConstructor(ctorDecl *BuildFunctionArgs) {
	resolvableTypesLock.Lock()
	constructorsDecls = append(constructorsDecls, ctorDecl)
	resolvableTypesLock.Unlock()
}

func newGoType(name string, zeroValue interface{}) px.ObjectType {
	obj := AllocObjectType()
	obj.name = name
	obj.initHashExpression = zeroValue
	registerResolvableType(obj)
	return obj
}

func registerResolvableType(tp px.ResolvableType) {
	resolvableTypesLock.Lock()
	resolvableTypes = append(resolvableTypes, tp)
	resolvableTypesLock.Unlock()
}

func registerMapping(t px.Type, r reflect.Type) {
	resolvableTypesLock.Lock()
	resolvableMappings = append(resolvableMappings, Mapping{t, r})
	resolvableTypesLock.Unlock()
}

func appendKey(b *bytes.Buffer, v px.Value) {
	if hk, ok := v.(px.StreamHashKeyValue); ok {
		hk.ToKey(b)
	} else if pt, ok := v.(px.Type); ok {
		b.WriteByte(1)
		b.WriteByte(HkType)
		b.Write([]byte(pt.Name()))
		if ppt, ok := pt.(px.ParameterizedType); ok {
			for _, p := range ppt.Parameters() {
				appendTypeParamKey(b, p)
			}
		}
	} else if hk, ok := v.(px.HashKeyValue); ok {
		b.Write([]byte(hk.ToKey()))
	} else {
		panic(illegalArgumentType(`ToKey`, 0, `value used as hash key`, v))
	}
}

// Special hash key generation for type parameters which might be hashes
// using string keys
func appendTypeParamKey(b *bytes.Buffer, v px.Value) {
	if h, ok := v.(*Hash); ok {
		b.WriteByte(2)
		h.EachPair(func(k, v px.Value) {
			b.Write([]byte(k.String()))
			b.WriteByte(3)
			appendTypeParamKey(b, v)
		})
	} else {
		appendKey(b, v)
	}
}

func wrap(c px.Context, v interface{}) (pv px.Value) {
	switch v := v.(type) {
	case nil:
		pv = undef
	case px.Value:
		pv = v
	case string:
		pv = stringValue(v)
	case int8:
		pv = integerValue(int64(v))
	case int16:
		pv = integerValue(int64(v))
	case int32:
		pv = integerValue(int64(v))
	case int64:
		pv = integerValue(v)
	case byte:
		pv = integerValue(int64(v))
	case int:
		pv = integerValue(int64(v))
	case float64:
		pv = floatValue(v)
	case bool:
		pv = booleanValue(v)
	case *regexp.Regexp:
		pv = WrapRegexp2(v)
	case []byte:
		pv = WrapBinary(v)
	case semver.Version:
		pv = WrapSemVer(v)
	case semver.VersionRange:
		pv = WrapSemVerRange(v)
	case time.Duration:
		pv = WrapTimespan(v)
	case time.Time:
		pv = WrapTimestamp(v)
	case []int:
		pv = WrapInts(v)
	case []string:
		pv = WrapStrings(v)
	case []px.Value:
		pv = WrapValues(v)
	case []px.Type:
		pv = WrapTypes(v)
	case []interface{}:
		return WrapInterfaces(c, v)
	case map[string]interface{}:
		pv = WrapStringToInterfaceMap(c, v)
	case map[string]string:
		pv = WrapStringToStringMap(v)
	case map[string]px.Value:
		pv = WrapStringToValueMap(v)
	case map[string]px.Type:
		pv = WrapStringToTypeMap(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			pv = integerValue(i)
		} else {
			f, _ := v.Float64()
			pv = floatValue(f)
		}
	case reflect.Value:
		pv = wrapReflected(c, v)
	case reflect.Type:
		var err error
		if pv, err = wrapReflectedType(c, v); err != nil {
			panic(err)
		}
	default:
		// Can still be an alias, slice, or map in which case reflection conversion will work
		pv = wrapReflected(c, reflect.ValueOf(v))
	}
	return pv
}

func wrapReflected(c px.Context, vr reflect.Value) (pv px.Value) {
	if c == nil {
		c = px.CurrentContext()
	}

	// Invalid shouldn't happen, but needs a check
	if !vr.IsValid() {
		return undef
	}

	vi := vr

	// Check for nil
	switch vr.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map, reflect.Interface:
		if vr.IsNil() {
			return undef
		}

		if vi.Kind() == reflect.Interface {
			// Need implementation here.
			vi = vi.Elem()
		}
	}

	vt := vr.Type()
	if _, ok := wellKnown[vt]; ok {
		iv := vr.Interface()
		if pv, ok = iv.(px.Value); ok {
			return
		}
		// A well-known that isn't an pcore.Value just yet
		return wrap(c, iv)
	}

	if t, ok := loadFromImplRegistry(c, vi.Type()); ok {
		if pt, ok := t.(px.ObjectType); ok {
			pv = pt.FromReflectedValue(c, vi)
			return
		}
	}

	pv, ok := WrapPrimitive(vr)
	if ok {
		return pv
	}

	switch vt.Kind() {
	case reflect.Slice, reflect.Array:
		top := vr.Len()
		els := make([]px.Value, top)
		for i := 0; i < top; i++ {
			els[i] = wrap(c, interfaceOrNil(vr.Index(i)))
		}
		pv = WrapValues(els)
	case reflect.Map:
		keys := vr.MapKeys()
		els := make([]*HashEntry, len(keys))
		for i, k := range keys {
			els[i] = WrapHashEntry(wrap(c, interfaceOrNil(k)), wrap(c, interfaceOrNil(vr.MapIndex(k))))
		}
		pv = sortedMap(els)
	case reflect.Ptr:
		pv = wrapReflected(c, vr.Elem())
	case reflect.Struct:
		// Unknown struct. Map this to a Hash<String,Any>
		nf := vt.NumField()
		els := make([]*HashEntry, 0, nf)
		for i := 0; i < nf; i++ {
			vv := vr.Field(i)
			switch vv.Kind() {
			case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
				if vv.IsNil() {
					continue
				}
			}
			ft := vt.Field(i)
			els = append(els, WrapHashEntry2(FieldName(&ft), wrap(c, interfaceOrNil(vv))))
		}
		pv = sortedMap(els)
	default:
		if vr.IsValid() && vr.CanInterface() {
			ix := vr.Interface()
			pv, ok = ix.(px.Value)
			if ok {
				return pv
			}
			pv = WrapRuntime(vr.Interface())
		} else {
			pv = undef
		}
	}
	return pv
}

func WrapPrimitive(vr reflect.Value) (pv px.Value, ok bool) {
	ok = true
	switch vr.Kind() {
	case reflect.String:
		pv = stringValue(vr.String())
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		pv = integerValue(vr.Int())
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		pv = integerValue(int64(vr.Uint())) // Possible loss for very large numbers
	case reflect.Bool:
		pv = booleanValue(vr.Bool())
	case reflect.Float64, reflect.Float32:
		pv = floatValue(vr.Float())
	default:
		ok = false
	}
	return
}

func loadFromImplRegistry(c px.Context, vt reflect.Type) (px.Type, bool) {
	if t, ok := c.ImplementationRegistry().ReflectedToType(vt); ok {
		return t, true
	}
	return nil, false
}

var evalValueType = reflect.TypeOf((*px.Value)(nil)).Elem()
var evalTypeType = reflect.TypeOf((*px.Type)(nil)).Elem()
var evalObjectTypeType = reflect.TypeOf((*px.ObjectType)(nil)).Elem()
var evalTypeSetType = reflect.TypeOf((*px.TypeSet)(nil)).Elem()

var wellKnown map[reflect.Type]px.Type

func wrapReflectedType(c px.Context, vt reflect.Type) (pt px.Type, err error) {
	if c == nil {
		c = px.CurrentContext()
	}

	var ok bool
	if pt, ok = wellKnown[vt]; ok {
		return
	}

	kind := vt.Kind()
	if pt, ok = loadFromImplRegistry(c, vt); ok {
		if kind == reflect.Ptr {
			pt = NewOptionalType(pt)
		}
		return
	}

	var t px.Type
	switch kind {
	case reflect.Slice, reflect.Array:
		if t, err = wrapReflectedType(c, vt.Elem()); err == nil {
			pt = NewArrayType(t, nil)
		}
	case reflect.Map:
		if t, err = wrapReflectedType(c, vt.Key()); err == nil {
			var v px.Type
			if v, err = wrapReflectedType(c, vt.Elem()); err == nil {
				pt = NewHashType(t, v, nil)
			}
		}
	case reflect.Ptr:
		if t, err = wrapReflectedType(c, vt.Elem()); err == nil {
			pt = NewOptionalType(t)
		}
	default:
		pt, ok = primitivePTypes[vt.Kind()]
		if !ok {
			err = px.Error(px.UnreflectableType, issue.H{`type`: vt.String()})
		}
	}
	return
}

var primitivePTypes map[reflect.Kind]px.Type

func PrimitivePType(vt reflect.Type) (pt px.Type, ok bool) {
	pt, ok = primitivePTypes[vt.Kind()]
	return
}

func interfaceOrNil(vr reflect.Value) interface{} {
	if vr.CanInterface() {
		return vr.Interface()
	}
	return nil
}

func newNamedType(name, typeDecl string) px.Type {
	dt, err := Parse(typeDecl)
	if err != nil {
		_, fileName, fileLine, _ := runtime.Caller(1)
		err = convertReported(err, fileName, fileLine)
		panic(err)
	}
	return NamedType(px.RuntimeNameAuthority, name, dt)
}

func convertReported(err error, fileName string, lineOffset int) error {
	if ri, ok := err.(issue.Reported); ok {
		return ri.OffsetByLocation(issue.NewLocation(fileName, lineOffset, 0))
	}
	return err
}

func NamedType(na px.URI, name string, value px.Value) px.Type {
	var ta px.Type
	if na == `` {
		na = px.RuntimeNameAuthority
	}
	if dt, ok := value.(*DeferredType); ok {
		if len(dt.params) == 1 {
			if hash, ok := dt.params[0].(px.OrderedMap); ok && dt.Name() != `Struct` {
				if dt.Name() == `Object` {
					ta = createMetaType2(na, name, dt.Name(), extractParentName2(hash), hash)
				} else if dt.Name() == `TypeSet` {
					ta = createMetaType2(na, name, dt.Name(), ``, hash)
				} else {
					ta = createMetaType2(na, name, `Object`, dt.Name(), hash)
				}
			}
		}
		if ta == nil {
			ta = NewTypeAliasType(name, dt, nil)
		}
	} else if h, ok := value.(px.OrderedMap); ok {
		ta = createMetaType2(na, name, `Object`, ``, h)
	} else {
		panic(fmt.Sprintf(`cannot create object from a %s`, dt.String()))
	}
	return ta
}

func extractParentName2(hash px.OrderedMap) string {
	if p, ok := hash.Get4(keyParent); ok {
		if dt, ok := p.(*DeferredType); ok {
			return dt.Name()
		}
		if s, ok := p.(px.StringValue); ok {
			return s.String()
		}
	}
	return ``
}

func createMetaType2(na px.URI, name string, typeName string, parentName string, hash px.OrderedMap) px.Type {
	if parentName == `` {
		switch typeName {
		case `Object`:
			return MakeObjectType(name, nil, hash, false)
		default:
			return NewTypeSet(na, name, hash)
		}
	}

	return MakeObjectType(name, NewTypeReferenceType(parentName), hash, false)
}

func argError(key string, e px.Type, a px.Value) issue.Reported {
	return px.Error(px.TypeMismatch, issue.H{`detail`: px.DescribeMismatch(`property '`+key+`'`, e, a.PType())})
}

func typeArg(hash px.OrderedMap, key string, d px.Type) px.Type {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(px.Type); ok {
		return t
	}
	panic(argError(key, DefaultTypeType(), v))
}

func hashArg(hash px.OrderedMap, key string) *Hash {
	v := hash.Get5(key, nil)
	if v == nil {
		return emptyMap
	}
	if t, ok := v.(*Hash); ok {
		return t
	}
	panic(argError(key, DefaultHashType(), v))
}

func boolArg(hash px.OrderedMap, key string, d bool) bool {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(booleanValue); ok {
		return t.Bool()
	}
	panic(argError(key, DefaultBooleanType(), v))
}

type LazyType interface {
	LazyIsInstance(v px.Value, g px.Guard) int
}

func LazyIsInstance(a px.Type, b px.Value, g px.Guard) int {
	if lt, ok := a.(LazyType); ok {
		return lt.LazyIsInstance(b, g)
	}
	if a.IsInstance(b, g) {
		return 1
	}
	return -1
}

func stringArg(hash px.OrderedMap, key string, d string) string {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(stringValue); ok {
		return string(t)
	}
	panic(argError(key, DefaultStringType(), v))
}

func uriArg(hash px.OrderedMap, key string, d px.URI) px.URI {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(stringValue); ok {
		str := string(t)
		if _, err := ParseURI2(str, true); err != nil {
			panic(px.Error(px.InvalidUri, issue.H{`str`: str, `detail`: err.Error()}))
		}
		return px.URI(str)
	}
	if t, ok := v.(*UriValue); ok {
		return px.URI(t.URL().String())
	}
	panic(argError(key, DefaultUriType(), v))
}

func versionArg(hash px.OrderedMap, key string, d semver.Version) semver.Version {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if s, ok := v.(stringValue); ok {
		sv, err := semver.ParseVersion(string(s))
		if err != nil {
			panic(px.Error(px.InvalidVersion, issue.H{`str`: string(s), `detail`: err.Error()}))
		}
		return sv
	}
	if sv, ok := v.(*SemVer); ok {
		return sv.Version()
	}
	panic(argError(key, DefaultSemVerType(), v))
}

func versionRangeArg(hash px.OrderedMap, key string, d semver.VersionRange) semver.VersionRange {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if s, ok := v.(stringValue); ok {
		sr, err := semver.ParseVersionRange(string(s))
		if err != nil {
			panic(px.Error(px.InvalidVersionRange, issue.H{`str`: string(s), `detail`: err.Error()}))
		}
		return sr
	}
	if sv, ok := v.(*SemVerRange); ok {
		return sv.VersionRange()
	}
	panic(argError(key, DefaultSemVerType(), v))
}
