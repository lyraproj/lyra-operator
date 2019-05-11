package types

import (
	"io"
	"sync"

	"github.com/lyraproj/pcore/utils"

	"github.com/lyraproj/pcore/px"
)

type (
	StructElement struct {
		name  string
		key   px.Type
		value px.Type
	}

	StructType struct {
		lock          sync.Mutex
		elements      []*StructElement
		hashedMembers map[string]*StructElement
	}
)

var StructElementMeta px.Type

var StructMetaType px.ObjectType

func init() {
	StructElementMeta = newObjectType(`Pcore::StructElement`,
		`{
	attributes => {
		key_type => Type,
    value_type => Type
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return NewStructElement(args[0], args[1].(px.Type))
		})

	StructMetaType = newObjectType(`Pcore::StructType`,
		`Pcore::AnyType {
	attributes => {
		elements => Array[Pcore::StructElement]
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newStructType2(args...)
		})

	// Go constructor for Struct instances is registered by HashType
}

func NewStructElement(key px.Value, value px.Type) *StructElement {

	var (
		name    string
		keyType px.Type
	)

	switch key := key.(type) {
	case stringValue:
		keyType = key.PType()
		if isAssignable(value, DefaultUndefType()) {
			keyType = NewOptionalType(keyType)
		}
		name = string(key)
	case *vcStringType:
		name = key.value
		keyType = key
	case *OptionalType:
		if strType, ok := key.typ.(*vcStringType); ok {
			name = strType.value
			keyType = key
		}
	}

	if keyType == nil || name == `` {
		panic(illegalArgumentType(`StructElement`, 0, `Variant[String[1], Type[String[1]], , Type[Optional[String[1]]]]`, key))
	}
	return &StructElement{name, keyType, value}
}

func newStructElement2(key string, value px.Type) *StructElement {
	return NewStructElement(stringValue(key), value)
}

func DefaultStructType() *StructType {
	return structTypeDefault
}

func NewStructType(elements []*StructElement) *StructType {
	if len(elements) == 0 {
		return DefaultStructType()
	}
	return &StructType{elements: elements}
}

func newStructType2(args ...px.Value) *StructType {
	switch len(args) {
	case 0:
		return DefaultStructType()
	case 1:
		arg := args[0]
		if ar, ok := arg.(*Array); ok {
			return newStructType2(ar.AppendTo(make([]px.Value, 0, ar.Len()))...)
		}
		hash, ok := arg.(px.OrderedMap)
		if !ok {
			panic(illegalArgumentType(`Struct[]`, 0, `Hash[Variant[String[1], Optional[String[1]]], Type]`, arg))
		}
		top := hash.Len()
		es := make([]*StructElement, top)
		hash.EachWithIndex(func(v px.Value, idx int) {
			e := v.(*HashEntry)
			vt, ok := e.Value().(px.Type)
			if !ok {
				panic(illegalArgumentType(`StructElement`, 1, `Type`, v))
			}
			es[idx] = NewStructElement(e.Key(), vt)
		})
		return NewStructType(es)
	default:
		panic(illegalArgumentCount(`Struct`, `0 - 1`, len(args)))
	}
}

func (s *StructElement) Accept(v px.Visitor, g px.Guard) {
	s.key.Accept(v, g)
	s.value.Accept(v, g)
}

func (s *StructElement) ActualKeyType() px.Type {
	if ot, ok := s.key.(*OptionalType); ok {
		return ot.typ
	}
	return s.key
}

func (s *StructElement) Equals(o interface{}, g px.Guard) bool {
	if ose, ok := o.(*StructElement); ok {
		return s.key.Equals(ose.key, g) && s.value.Equals(ose.value, g)
	}
	return false
}

func (s *StructElement) String() string {
	return px.ToString(s)
}

func (s *StructElement) PType() px.Type {
	return StructElementMeta
}

func (s *StructElement) Key() px.Type {
	return s.key
}

func (s *StructElement) Name() string {
	return s.name
}

func (s *StructElement) Optional() bool {
	_, ok := s.key.(*OptionalType)
	return ok
}

func (s *StructElement) resolve(c px.Context) {
	s.key = resolve(c, s.key)
	s.value = resolve(c, s.value)
}

func (s *StructElement) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	optionalValue := isAssignable(s.value, undefTypeDefault)
	if _, ok := s.key.(*OptionalType); ok {
		if optionalValue {
			utils.WriteString(bld, s.name)
		} else {
			s.key.ToString(bld, format, g)
		}
	} else {
		if optionalValue {
			NewNotUndefType(s.key).ToString(bld, format, g)
		} else {
			utils.WriteString(bld, s.name)
		}
	}
	utils.WriteString(bld, ` => `)
	s.value.ToString(bld, format, g)
}

func (s *StructElement) Value() px.Type {
	return s.value
}

func (t *StructType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	for _, element := range t.elements {
		element.Accept(v, g)
	}
}

func (t *StructType) Default() px.Type {
	return structTypeDefault
}

func (t *StructType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*StructType); ok && len(t.elements) == len(ot.elements) {
		for idx, element := range t.elements {
			if !element.Equals(ot.elements[idx], g) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *StructType) Generic() px.Type {
	al := make([]*StructElement, len(t.elements))
	for idx, e := range t.elements {
		al[idx] = &StructElement{e.name, px.GenericType(e.key), px.GenericType(e.value)}
	}
	return NewStructType(al)
}

func (t *StructType) Elements() []*StructElement {
	return t.elements
}

func (t *StructType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `elements`:
		els := make([]px.Value, len(t.elements))
		for i, e := range t.elements {
			els[i] = e
		}
		return WrapValues(els), true
	}
	return nil, false
}

func (t *StructType) HashedMembers() map[string]*StructElement {
	t.lock.Lock()
	if t.hashedMembers == nil {
		t.hashedMembers = t.HashedMembersCloned()
	}
	t.lock.Unlock()
	return t.hashedMembers
}

func (t *StructType) HashedMembersCloned() map[string]*StructElement {
	hashedMembers := make(map[string]*StructElement, len(t.elements))
	for _, elem := range t.elements {
		hashedMembers[elem.name] = elem
	}
	return hashedMembers
}

func (t *StructType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o := o.(type) {
	case *StructType:
		hm := o.HashedMembers()
		matched := 0
		for _, e1 := range t.elements {
			e2 := hm[e1.name]
			if e2 == nil {
				if !GuardedIsAssignable(e1.key, undefTypeDefault, g) {
					return false
				}
			} else {
				if !(GuardedIsAssignable(e1.key, e2.key, g) && GuardedIsAssignable(e1.value, e2.value, g)) {
					return false
				}
				matched++
			}
		}
		return matched == len(hm)
	case *HashType:
		required := 0
		for _, e := range t.elements {
			if !GuardedIsAssignable(e.key, undefTypeDefault, g) {
				if !GuardedIsAssignable(e.value, o.valueType, g) {
					return false
				}
				required++
			}
		}
		if required > 0 && !GuardedIsAssignable(stringTypeDefault, o.keyType, g) {
			return false
		}
		return GuardedIsAssignable(NewIntegerType(int64(required), int64(len(t.elements))), o.size, g)
	default:
		return false
	}
}

func (t *StructType) IsInstance(o px.Value, g px.Guard) bool {
	ov, ok := o.(*Hash)
	if !ok {
		return false
	}
	matched := 0
	for _, element := range t.elements {
		key := element.name
		v, ok := ov.Get(stringValue(key))
		if !ok {
			if !GuardedIsAssignable(element.key, undefTypeDefault, g) {
				return false
			}
		} else {
			if !GuardedIsInstance(element.value, v, g) {
				return false
			}
			matched++
		}
	}
	return matched == ov.Len()
}

func (t *StructType) MetaType() px.ObjectType {
	return StructMetaType
}

func (t *StructType) Name() string {
	return `Struct`
}

func (t *StructType) Parameters() []px.Value {
	top := len(t.elements)
	if top == 0 {
		return px.EmptyValues
	}
	entries := make([]*HashEntry, top)
	for idx, s := range t.elements {
		optionalValue := isAssignable(s.value, undefTypeDefault)
		var key px.Value
		if _, ok := s.key.(*OptionalType); ok {
			if optionalValue {
				key = stringValue(s.name)
			} else {
				key = s.key
			}
		} else {
			if optionalValue {
				key = NewNotUndefType(s.key)
			} else {
				key = stringValue(s.name)
			}
		}
		entries[idx] = WrapHashEntry(key, s.value)
	}
	return []px.Value{WrapHash(entries)}
}

func (t *StructType) Resolve(c px.Context) px.Type {
	for _, e := range t.elements {
		e.resolve(c)
	}
	return t
}

func (t *StructType) CanSerializeAsString() bool {
	for _, v := range t.elements {
		if !(canSerializeAsString(v.key) && canSerializeAsString(v.value)) {
			return false
		}
	}
	return true
}

func (t *StructType) SerializationString() string {
	return t.String()
}

func (t *StructType) Size() *IntegerType {
	required := 0
	for _, e := range t.elements {
		if !GuardedIsAssignable(e.key, undefTypeDefault, nil) {
			required++
		}
	}
	return NewIntegerType(int64(required), int64(len(t.elements)))
}

func (t *StructType) String() string {
	return px.ToString2(t, None)
}

func (t *StructType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *StructType) PType() px.Type {
	return &TypeType{t}
}

var structTypeDefault = &StructType{elements: []*StructElement{}}
