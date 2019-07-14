package types

import (
	"math"
	"reflect"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/semver/semver"
)

const tagName = "puppet"

type reflector struct {
	c px.Context
}

var pValueType = reflect.TypeOf((*px.Value)(nil)).Elem()

func NewReflector(c px.Context) px.Reflector {
	return &reflector{c}
}

func Methods(t reflect.Type) []reflect.Method {
	if t.Kind() == reflect.Ptr {
		// Pointer may have methods
		if t.NumMethod() == 0 {
			t = t.Elem()
		}
	}
	nm := t.NumMethod()
	ms := make([]reflect.Method, nm)
	for i := 0; i < nm; i++ {
		ms[i] = t.Method(i)
	}
	return ms
}

func Fields(t reflect.Type) []reflect.StructField {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	nf := 0
	if t.Kind() == reflect.Struct {
		nf = t.NumField()
	}
	fs := make([]reflect.StructField, nf)
	for i := 0; i < nf; i++ {
		fs[i] = t.Field(i)
	}
	return fs
}

// NormalizeType ensures that pointers to interface is converted to interface and that struct is converted to
// pointer to struct
func NormalizeType(rt reflect.Type) reflect.Type {
	switch rt.Kind() {
	case reflect.Struct:
		rt = reflect.PtrTo(rt)
	case reflect.Ptr:
		re := rt.Elem()
		if re.Kind() == reflect.Interface {
			rt = re
		}
	}
	return rt
}

func (r *reflector) Methods(t reflect.Type) []reflect.Method {
	return Methods(t)
}

func (r *reflector) Fields(t reflect.Type) []reflect.StructField {
	return Fields(t)
}

func (r *reflector) FieldName(f *reflect.StructField) string {
	return FieldName(f)
}

func FieldName(f *reflect.StructField) string {
	if tagHash, ok := TagHash(f); ok {
		if nv, ok := tagHash.Get4(`name`); ok {
			return nv.String()
		}
	}
	return issue.FirstToLower(f.Name)
}

func (r *reflector) Reflect(src px.Value) reflect.Value {
	if sn, ok := src.(px.Reflected); ok {
		return sn.Reflect(r.c)
	}
	panic(px.Error(px.UnreflectableValue, issue.H{`type`: src.PType()}))
}

func (r *reflector) Reflect2(src px.Value, rt reflect.Type) reflect.Value {
	if rt != nil && rt.Kind() == reflect.Interface && rt.AssignableTo(pValueType) {
		sv := reflect.ValueOf(src)
		if sv.Type().AssignableTo(rt) {
			return sv
		}
	}
	v := reflect.New(rt).Elem()
	r.ReflectTo(src, v)
	return v
}

// ReflectTo assigns the native value of src to dest
func (r *reflector) ReflectTo(src px.Value, dest reflect.Value) {
	dt := dest.Type()
	assertSettable(&dest)
	if dt.Kind() == reflect.Interface && dt.AssignableTo(pValueType) {
		sv := reflect.ValueOf(src)
		if !sv.Type().AssignableTo(dt) {
			panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: sv.Type().String(), `actual`: dest.Type().String()}))
		}
		dest.Set(sv)
	} else {
		switch src := src.(type) {
		case px.Reflected:
			if dt.Kind() == reflect.Interface && dt.Name() == `` {
				// Destination is an interface{}, derive type from source
				dest.Set(src.Reflect(r.c))
			} else {
				src.ReflectTo(r.c, dest)
			}
		case px.PuppetObject:
			src.PType().(px.ObjectType).ToReflectedValue(r.c, src, dest)
		default:
			panic(px.Error(px.InvalidSourceForSet, issue.H{`type`: src.PType()}))
		}
	}
}

func (r *reflector) ReflectType(src px.Type) (reflect.Type, bool) {
	return ReflectType(r.c, src)
}

func ReflectType(c px.Context, src px.Type) (reflect.Type, bool) {
	if sn, ok := src.(px.ReflectedType); ok {
		return sn.ReflectType(c)
	}
	return nil, false
}

func (r *reflector) TagHash(f *reflect.StructField) (px.OrderedMap, bool) {
	return TagHash(f)
}

func TagHash(f *reflect.StructField) (px.OrderedMap, bool) {
	return ParseTagHash(f.Tag.Get(tagName))
}

func ParseTagHash(tag string) (px.OrderedMap, bool) {
	if tag != `` {
		tagExpr, err := Parse(`{` + tag + `}`)
		if err == nil {
			return tagExpr.(px.OrderedMap), true
		}
	}
	return nil, false
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (r *reflector) FunctionDeclFromReflect(name string, mt reflect.Type, withReceiver bool) px.OrderedMap {
	returnsError := false
	var rt px.Type
	var err error
	oc := mt.NumOut()
	switch oc {
	case 0:
		rt = DefaultAnyType()
	case 1:
		ot := mt.Out(0)
		if ot.AssignableTo(errorType) {
			returnsError = true
		} else {
			rt, err = wrapReflectedType(r.c, mt.Out(0))
			if err != nil {
				panic(err)
			}
		}
	case 2:
		rt, err = wrapReflectedType(r.c, mt.Out(0))
		if err != nil {
			panic(err)
		}
		ot := mt.Out(1)
		if ot.AssignableTo(errorType) {
			returnsError = true
		} else {
			var rt2 px.Type
			rt2, err = wrapReflectedType(r.c, mt.Out(1))
			if err != nil {
				panic(err)
			}
			rt = NewTupleType([]px.Type{rt, rt2}, nil)
		}
	default:
		ot := mt.Out(oc - 1)
		if ot.AssignableTo(errorType) {
			returnsError = true
			oc = oc - 1
		}
		ts := make([]px.Type, oc)
		for i := 0; i < oc; i++ {
			ts[i], err = wrapReflectedType(r.c, mt.Out(i))
			if err != nil {
				panic(err)
			}
		}
		rt = NewTupleType(ts, nil)
	}

	var pt *TupleType
	pc := mt.NumIn()
	ix := 0
	if withReceiver {
		// First argument is the receiver itself
		ix = 1
	}

	if pc == ix {
		pt = EmptyTupleType()
	} else {
		ps := make([]px.Type, pc-ix)
		for p := ix; p < pc; p++ {
			ps[p-ix], err = wrapReflectedType(r.c, mt.In(p))
			if err != nil {
				panic(err)
			}
		}
		var sz *IntegerType
		if mt.IsVariadic() {
			last := pc - ix - 1
			ps[last] = ps[last].(*ArrayType).ElementType()
			sz = NewIntegerType(int64(last), math.MaxInt64)
		}
		pt = NewTupleType(ps, sz)
	}
	ne := 2
	if returnsError {
		ne++
	}
	ds := make([]*HashEntry, ne)
	ds[0] = WrapHashEntry2(keyType, NewCallableType(pt, rt, nil))
	ds[1] = WrapHashEntry2(KeyGoName, stringValue(name))
	if returnsError {
		ds[2] = WrapHashEntry2(keyReturnsError, BooleanTrue)
	}
	return WrapHash(ds)
}

func (r *reflector) InitializerFromTagged(typeName string, parent px.Type, tg px.AnnotatedType) px.OrderedMap {
	rf := tg.Type()
	ie := make([]*HashEntry, 0, 2)
	if rf.Kind() == reflect.Func {
		fn := rf.Name()
		if fn == `` {
			fn = `do`
		}
		ie = append(ie, WrapHashEntry2(keyFunctions, singletonMap(`do`, r.FunctionDeclFromReflect(fn, rf, false))))
	} else {
		tags := tg.Tags()
		otherTags := tg.OtherTags()
		fs := r.Fields(rf)
		nf := len(fs)
		var pt reflect.Type

		if nf > 0 {
			es := make([]*HashEntry, 0, nf)
			for i, f := range fs {
				if i == 0 && f.Anonymous {
					// Parent
					pt = reflect.PtrTo(f.Type)
					continue
				}
				if f.PkgPath != `` {
					// Unexported
					continue
				}

				name, decl := r.ReflectFieldTags(&f, tags[f.Name], otherTags[f.Name])
				es = append(es, WrapHashEntry2(name, decl))
			}
			ie = append(ie, WrapHashEntry2(keyAttributes, WrapHash(es)))
		}

		ms := r.Methods(rf)
		nm := len(ms)
		if nm > 0 {
			es := make([]*HashEntry, 0, nm)
			for _, m := range ms {
				if m.PkgPath != `` {
					// Not exported struct method
					continue
				}

				if pt != nil {
					if _, ok := pt.MethodByName(m.Name); ok {
						// Redeclaration's of parent method are not included
						continue
					}
				}
				es = append(es, WrapHashEntry2(issue.FirstToLower(m.Name), r.FunctionDeclFromReflect(m.Name, m.Type, rf.Kind() != reflect.Interface)))
			}
			ie = append(ie, WrapHashEntry2(keyFunctions, WrapHash(es)))
		}
	}
	ats := tg.Annotations()
	if ats != nil && !ats.IsEmpty() {
		ie = append(ie, WrapHashEntry2(keyAnnotations, ats))
	}
	return WrapHash(ie)
}

func (r *reflector) TypeFromReflect(typeName string, parent px.Type, rf reflect.Type) px.ObjectType {
	return r.TypeFromTagged(typeName, parent, px.NewTaggedType(rf, nil), nil)
}

func (r *reflector) TypeFromTagged(typeName string, parent px.Type, tg px.AnnotatedType, rcFunc px.Doer) px.ObjectType {
	return BuildObjectType(typeName, parent, func(obj px.ObjectType) px.OrderedMap {
		obj.(*objectType).goType = tg

		r.c.ImplementationRegistry().RegisterType(obj, tg.Type())
		if rcFunc != nil {
			rcFunc()
		}
		return r.InitializerFromTagged(typeName, parent, tg)
	})
}

func (r *reflector) ReflectFieldTags(f *reflect.StructField, fh px.OrderedMap, otherTags map[string]string) (name string, decl px.OrderedMap) {
	as := make([]*HashEntry, 0)
	var val px.Value
	var typ px.Type

	if fh != nil {
		if v, ok := fh.Get4(keyName); ok {
			name = v.String()
		}
		if v, ok := fh.GetEntry(keyKind); ok {
			as = append(as, v.(*HashEntry))
		}
		if v, ok := fh.GetEntry(keyValue); ok {
			val = v.Value()
			as = append(as, v.(*HashEntry))
		}
		if v, ok := fh.Get4(keyType); ok {
			switch v := v.(type) {
			case *DeferredType:
				typ = v.Resolve(r.c)
			case px.Type:
				typ = v
			}
		}
	}

	if typ == nil {
		var err error
		if typ, err = px.WrapReflectedType(r.c, f.Type); err != nil {
			panic(err)
		}
	}

	optional := typ.IsInstance(px.Undef, nil)
	if optional {
		if val == nil {
			// If no value is declared and the type is declared as optional, then
			// value is an implicit undef
			as = append(as, WrapHashEntry2(keyValue, undef))
		}
	} else {
		if val != nil && val.Equals(undef, nil) {
			// Convenience. If a value is declared as being undef, then ensure that
			// type accepts undef
			typ = NewOptionalType(typ)
			optional = true
		}
	}

	if optional {
		switch f.Type.Kind() {
		case reflect.Ptr, reflect.Interface:
			// OK. Can be nil
		default:
			// The field will always have a value (the Go zero value), so it cannot be nil.
			panic(px.Error(px.ImpossibleOptional, issue.H{`name`: f.Name, `type`: typ.String()}))
		}
	}

	as = append(as, WrapHashEntry2(keyType, typ))
	as = append(as, WrapHashEntry2(KeyGoName, stringValue(f.Name)))
	if len(otherTags) > 0 {
		as = append(as, WrapHashEntry2(keyAnnotations, singleMap(TagsAnnotationType, WrapStringToStringMap(otherTags))))
	}
	if name == `` {
		name = issue.FirstToLower(f.Name)
	}
	return name, WrapHash(as)
}

func (r *reflector) TypeSetFromReflect(typeSetName string, version semver.Version, aliases map[string]string, rTypes ...reflect.Type) px.TypeSet {
	types := make([]*HashEntry, 0)
	prefix := typeSetName + `::`
	for _, rt := range rTypes {
		var parent px.Type
		fs := r.Fields(rt)
		nf := len(fs)
		if nf > 0 {
			f := fs[0]
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				parent = NewTypeReferenceType(typeName(prefix, aliases, f.Type))
			}
		}
		name := typeName(prefix, aliases, rt)
		types = append(types, WrapHashEntry2(
			name[strings.LastIndex(name, `::`)+2:],
			r.TypeFromReflect(name, parent, rt)))
	}

	es := make([]*HashEntry, 0)
	es = append(es, WrapHashEntry2(px.KeyPcoreUri, stringValue(string(px.PcoreUri))))
	es = append(es, WrapHashEntry2(px.KeyPcoreVersion, WrapSemVer(px.PcoreVersion)))
	es = append(es, WrapHashEntry2(KeyVersion, WrapSemVer(version)))
	es = append(es, WrapHashEntry2(KeyTypes, WrapHash(types)))
	return NewTypeSet(px.RuntimeNameAuthority, typeSetName, WrapHash(es))
}

func ParentType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct && t.NumField() > 0 {
		f := t.Field(0)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			return f.Type
		}
	}
	return nil
}

func typeName(prefix string, aliases map[string]string, rt reflect.Type) string {
	if rt.Kind() == reflect.Ptr {
		// Pointers have no names
		rt = rt.Elem()
	}
	name := rt.Name()
	if aliases != nil {
		if alias, ok := aliases[name]; ok {
			name = alias
		}
	}
	return prefix + name
}

func assertSettable(value *reflect.Value) {
	if !value.CanSet() {
		panic(px.Error(px.AttemptToSetUnsettable, issue.H{`kind`: value.Type().String()}))
	}
}
