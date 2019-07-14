package types

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type (
	HashType struct {
		size      *IntegerType
		keyType   px.Type
		valueType px.Type
	}

	HashEntry struct {
		key   px.Value
		value px.Value
	}

	Hash struct {
		reducedType  *HashType
		detailedType px.Type
		entries      []*HashEntry
		index        map[px.HashKey]int
	}

	MutableHashValue struct {
		Hash
	}
)

var hashTypeEmpty = &HashType{IntegerTypeZero, unitTypeDefault, unitTypeDefault}
var hashTypeDefault = &HashType{IntegerTypePositive, anyTypeDefault, anyTypeDefault}

var HashMetaType px.ObjectType

func init() {
	HashMetaType = newObjectType(`Pcore::HashType`,
		`Pcore::CollectionType {
	attributes => {
		key_type => {
			type => Optional[Type],
			value => Any
		},
		value_type => {
			type => Optional[Type],
			value => Any
		},
	},
  serialization => [ 'key_type', 'value_type', 'size_type' ]
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newHashType2(args...)
		},
		func(ctx px.Context, args []px.Value) px.Value {
			h := args[0].(*Hash)
			kt := h.Get5(`key_type`, DefaultAnyType())
			vt := h.Get5(`value_type`, DefaultAnyType())
			st := h.Get5(`size_type`, PositiveIntegerType())
			return newHashType2(kt, vt, st)
		})

	newGoConstructor3([]string{`Hash`, `Struct`},
		func(t px.LocalTypes) {
			t.Type(`KeyValueArray`, `Array[Tuple[Any,Any],1]`)
			t.Type(`TreeArray`, `Array[Tuple[Array,Any],1]`)
			t.Type(`NewHashOption`, `Enum[tree, hash_tree]`)
		},

		func(d px.Dispatch) {
			d.Param(`TreeArray`)
			d.OptionalParam(`NewHashOption`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				if len(args) < 2 {
					return WrapHashFromArray(args[0].(*Array))
				}
				allHashes := args[1].String() == `hash_tree`
				result := NewMutableHash()
				args[0].(*Array).Each(func(entry px.Value) {
					tpl := entry.(*Array)
					path := tpl.At(0).(*Array)
					value := tpl.At(1)
					if path.IsEmpty() {
						// root node (index [] was included - values merge into the result)
						//  An array must be changed to a hash first as this is the root
						// (Cannot return an array from a Hash.new)
						if av, ok := value.(*Array); ok {
							result.PutAll(IndexedFromArray(av))
						} else {
							if hv, ok := value.(px.OrderedMap); ok {
								result.PutAll(hv)
							}
						}
					} else {
						r := path.Slice(0, path.Len()-1).Reduce2(result, func(memo, idx px.Value) px.Value {
							if hv, ok := memo.(*MutableHashValue); ok {
								return hv.Get3(idx, func() px.Value {
									x := NewMutableHash()
									hv.Put(idx, x)
									return x
								})
							}
							if av, ok := memo.(px.List); ok {
								if ix, ok := idx.(integerValue); ok {
									return av.At(int(ix))
								}
							}
							return undef
						})
						if hr, ok := r.(*MutableHashValue); ok {
							if allHashes {
								if av, ok := value.(*Array); ok {
									value = IndexedFromArray(av)
								}
							}
							hr.Put(path.At(path.Len()-1), value)
						}
					}
				})
				return &result.Hash
			})
		},

		func(d px.Dispatch) {
			d.Param(`KeyValueArray`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return WrapHashFromArray(args[0].(*Array))
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				switch arg := args[0].(type) {
				case *Array:
					return WrapHashFromArray(arg)
				case *Hash:
					return arg
				default:
					return WrapHashFromArray(arg.(px.Arrayable).AsArray().(*Array))
				}
			})
		},
	)
	px.SingletonMap = singletonMap
}

func DefaultHashType() *HashType {
	return hashTypeDefault
}

func EmptyHashType() *HashType {
	return hashTypeEmpty
}

func NewHashType(keyType px.Type, valueType px.Type, rng *IntegerType) *HashType {
	if rng == nil {
		rng = IntegerTypePositive
	}
	if keyType == nil {
		keyType = anyTypeDefault
	}
	if valueType == nil {
		valueType = anyTypeDefault
	}
	if keyType == anyTypeDefault && valueType == anyTypeDefault && rng == IntegerTypePositive {
		return DefaultHashType()
	}
	return &HashType{rng, keyType, valueType}
}

func newHashType2(args ...px.Value) *HashType {
	argc := len(args)
	if argc == 0 {
		return hashTypeDefault
	}

	if argc == 1 || argc > 4 {
		panic(illegalArgumentCount(`Hash[]`, `0, 2, or 3`, argc))
	}

	offset := 0
	var valueType px.Type
	keyType, ok := args[0].(px.Type)
	if ok {
		valueType, ok = args[1].(px.Type)
		if !ok {
			panic(illegalArgumentType(`Hash[]`, 1, `Type`, args[1]))
		}
		offset += 2
	} else {
		keyType = DefaultAnyType()
		valueType = DefaultAnyType()
	}

	var rng *IntegerType
	switch argc - offset {
	case 0:
		rng = IntegerTypePositive
	case 1:
		sizeArg := args[offset]
		if rng, ok = sizeArg.(*IntegerType); !ok {
			var sz int64
			if sz, ok = toInt(sizeArg); !ok {
				panic(illegalArgumentType(`Hash[]`, offset, `Integer or Type[Integer]`, args[2]))
			}
			rng = NewIntegerType(sz, math.MaxInt64)
		}
	case 2:
		var min, max int64
		if min, ok = toInt(args[offset]); !ok {
			panic(illegalArgumentType(`Hash[]`, offset, `Integer`, args[offset]))
		}
		if max, ok = toInt(args[offset+1]); !ok {
			panic(illegalArgumentType(`Hash[]`, offset+1, `Integer`, args[offset+1]))
		}
		if min == 0 && max == 0 && offset == 0 {
			return hashTypeEmpty
		}
		rng = NewIntegerType(min, max)
	}
	return NewHashType(keyType, valueType, rng)
}

func (t *HashType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.size.Accept(v, g)
	t.keyType.Accept(v, g)
	t.valueType.Accept(v, g)
}

func (t *HashType) Default() px.Type {
	return hashTypeDefault
}

func (t *HashType) EntryType() px.Type {
	return NewTupleType([]px.Type{t.keyType, t.valueType}, nil)
}

func (t *HashType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*HashType); ok {
		return t.size.Equals(ot.size, g) && t.keyType.Equals(ot.keyType, g) && t.valueType.Equals(ot.valueType, g)
	}
	return false
}

func (t *HashType) Generic() px.Type {
	return NewHashType(px.GenericType(t.keyType), px.GenericType(t.valueType), nil)
}

func (t *HashType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `key_type`:
		return t.keyType, true
	case `value_type`:
		return t.valueType, true
	case `size_type`:
		return t.size, true
	}
	return nil, false
}

func (t *HashType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o := o.(type) {
	case *HashType:
		if t.size.min == 0 && o == hashTypeEmpty {
			return true
		}
		return t.size.IsAssignable(o.size, g) && GuardedIsAssignable(t.keyType, o.keyType, g) && GuardedIsAssignable(t.valueType, o.valueType, g)
	case *StructType:
		if !t.size.IsInstance3(len(o.elements)) {
			return false
		}
		for _, element := range o.elements {
			if !(GuardedIsAssignable(t.keyType, element.ActualKeyType(), g) && GuardedIsAssignable(t.valueType, element.value, g)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (t *HashType) IsInstance(o px.Value, g px.Guard) bool {
	if v, ok := o.(*Hash); ok && t.size.IsInstance3(v.Len()) {
		for _, entry := range v.entries {
			if !(GuardedIsInstance(t.keyType, entry.key, g) && GuardedIsInstance(t.valueType, entry.value, g)) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *HashType) KeyType() px.Type {
	return t.keyType
}

func (t *HashType) MetaType() px.ObjectType {
	return HashMetaType
}

func (t *HashType) Name() string {
	return `Hash`
}

func (t *HashType) Parameters() []px.Value {
	if *t == *hashTypeDefault {
		return px.EmptyValues
	}
	if *t == *hashTypeEmpty {
		return []px.Value{ZERO, ZERO}
	}
	params := make([]px.Value, 0, 4)
	params = append(params, t.keyType)
	params = append(params, t.valueType)
	if *t.size != *IntegerTypePositive {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *HashType) ReflectType(c px.Context) (reflect.Type, bool) {
	if kt, ok := ReflectType(c, t.keyType); ok {
		if vt, ok := ReflectType(c, t.valueType); ok {
			return reflect.MapOf(kt, vt), true
		}
	}
	return nil, false
}

func (t *HashType) Resolve(c px.Context) px.Type {
	t.keyType = resolve(c, t.keyType)
	t.valueType = resolve(c, t.valueType)
	return t
}

func (t *HashType) CanSerializeAsString() bool {
	return canSerializeAsString(t.keyType) && canSerializeAsString(t.valueType)
}

func (t *HashType) SerializationString() string {
	return t.String()
}

func (t *HashType) Size() *IntegerType {
	return t.size
}

func (t *HashType) String() string {
	return px.ToString2(t, None)
}

func (t *HashType) ValueType() px.Type {
	return t.valueType
}

func (t *HashType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *HashType) PType() px.Type {
	return &TypeType{t}
}

func WrapHashEntry(key px.Value, value px.Value) *HashEntry {
	if key == nil {
		panic(px.Error(px.NilHashKey, issue.NoArgs))
	}
	if value == nil {
		panic(px.Error(px.NilHashValue, issue.H{`key`: key}))
	}
	return &HashEntry{key, value}
}

func WrapHashEntry2(key string, value px.Value) *HashEntry {
	if value == nil {
		panic(px.Error(px.NilHashValue, issue.H{`key`: key}))
	}
	return &HashEntry{stringValue(key), value}
}

func (he *HashEntry) Add(v px.Value) px.List {
	panic(`Operation not supported`)
}

func (he *HashEntry) AddAll(v px.List) px.List {
	panic(`Operation not supported`)
}

func (he *HashEntry) All(predicate px.Predicate) bool {
	return predicate(he.key) && predicate(he.value)
}

func (he *HashEntry) Any(predicate px.Predicate) bool {
	return predicate(he.key) || predicate(he.value)
}

func (he *HashEntry) AppendTo(slice []px.Value) []px.Value {
	return append(slice, he.key, he.value)
}

func (he *HashEntry) At(i int) px.Value {
	switch i {
	case 0:
		return he.key
	case 1:
		return he.value
	default:
		return undef
	}
}

func (he *HashEntry) Delete(v px.Value) px.List {
	panic(`Operation not supported`)
}

func (he *HashEntry) DeleteAll(v px.List) px.List {
	panic(`Operation not supported`)
}

func (he *HashEntry) DetailedType() px.Type {
	return NewTupleType([]px.Type{px.DetailedValueType(he.key), px.DetailedValueType(he.value)}, NewIntegerType(2, 2))
}

func (he *HashEntry) Each(consumer px.Consumer) {
	consumer(he.key)
	consumer(he.value)
}

func (he *HashEntry) EachSlice(n int, consumer px.SliceConsumer) {
	if n == 1 {
		consumer(SingletonArray(he.key))
		consumer(SingletonArray(he.value))
	} else if n >= 2 {
		consumer(he)
	}
}

func (he *HashEntry) EachWithIndex(consumer px.IndexedConsumer) {
	consumer(he.key, 0)
	consumer(he.value, 1)
}

func (he *HashEntry) ElementType() px.Type {
	return commonType(he.key.PType(), he.value.PType())
}

func (he *HashEntry) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*HashEntry); ok {
		return he.key.Equals(ov.key, g) && he.value.Equals(ov.value, g)
	}
	if iv, ok := o.(*Array); ok && iv.Len() == 2 {
		return he.key.Equals(iv.At(0), g) && he.value.Equals(iv.At(1), g)
	}
	return false
}

func (he *HashEntry) Find(predicate px.Predicate) (px.Value, bool) {
	if predicate(he.key) {
		return he.key, true
	}
	if predicate(he.value) {
		return he.value, true
	}
	return nil, false
}

func (he *HashEntry) Flatten() px.List {
	return he.AsArray().Flatten()
}

func (he *HashEntry) IsEmpty() bool {
	return false
}

func (he *HashEntry) IsHashStyle() bool {
	return false
}

func (he *HashEntry) AsArray() px.List {
	return WrapValues([]px.Value{he.key, he.value})
}

func (he *HashEntry) Key() px.Value {
	return he.key
}

func (he *HashEntry) Len() int {
	return 2
}

func (he *HashEntry) Map(mapper px.Mapper) px.List {
	return WrapValues([]px.Value{mapper(he.key), mapper(he.value)})
}

func (he *HashEntry) Select(predicate px.Predicate) px.List {
	if predicate(he.key) {
		if predicate(he.value) {
			return he
		}
		return SingletonArray(he.key)
	}
	if predicate(he.value) {
		return SingletonArray(he.value)
	}
	return px.EmptyArray
}

func (he *HashEntry) Slice(i int, j int) px.List {
	if i > 1 || i >= j {
		return px.EmptyArray
	}
	if i == 1 {
		return SingletonArray(he.value)
	}
	if j == 1 {
		return SingletonArray(he.key)
	}
	return he
}

func (he *HashEntry) Reduce(redactor px.BiMapper) px.Value {
	return redactor(he.key, he.value)
}

func (he *HashEntry) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return redactor(redactor(initialValue, he.key), he.value)
}

func (he *HashEntry) Reject(predicate px.Predicate) px.List {
	if predicate(he.key) {
		if predicate(he.value) {
			return px.EmptyArray
		}
		return SingletonArray(he.value)
	}
	if predicate(he.value) {
		return SingletonArray(he.key)
	}
	return he
}

func (he *HashEntry) PType() px.Type {
	return NewArrayType(commonType(he.key.PType(), he.value.PType()), NewIntegerType(2, 2))
}

func (he *HashEntry) Unique() px.List {
	if he.key.Equals(he.value, nil) {
		return SingletonArray(he.key)
	}
	return he
}

func (he *HashEntry) Value() px.Value {
	return he.value
}

func (he *HashEntry) String() string {
	return px.ToString2(he, None)
}

func (he *HashEntry) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	WrapValues([]px.Value{he.key, he.value}).ToString(b, s, g)
}

func BuildHash(len int, bld func(*Hash, []*HashEntry) []*HashEntry) *Hash {
	h := &Hash{entries: make([]*HashEntry, 0, len)}
	h.entries = bld(h, h.entries)
	return h
}

func WrapHash(entries []*HashEntry) *Hash {
	return &Hash{entries: entries}
}

func WrapHash2(entries px.List) *Hash {
	hvEntries := make([]*HashEntry, entries.Len())
	entries.EachWithIndex(func(entry px.Value, idx int) {
		hvEntries[idx] = entry.(*HashEntry)
	})
	return &Hash{entries: hvEntries}
}

// WrapStringToTypeMap builds an ordered map from adds all entries in the given map
func WrapStringToTypeMap(hash map[string]px.Type) *Hash {
	hvEntries := make([]*HashEntry, len(hash))
	i := 0
	for k, v := range hash {
		hvEntries[i] = WrapHashEntry(stringValue(k), v)
		i++
	}
	return sortedMap(hvEntries)
}

// WrapStringToValueMap builds an ordered map from adds all entries in the given map
func WrapStringToValueMap(hash map[string]px.Value) *Hash {
	hvEntries := make([]*HashEntry, len(hash))
	i := 0
	for k, v := range hash {
		hvEntries[i] = WrapHashEntry(stringValue(k), v)
		i++
	}
	return sortedMap(hvEntries)
}

// WrapStringToInterfaceMap does not preserve order since order is undefined in a Go map
func WrapStringToInterfaceMap(c px.Context, hash map[string]interface{}) *Hash {
	hvEntries := make([]*HashEntry, len(hash))
	i := 0
	for k, v := range hash {
		hvEntries[i] = WrapHashEntry2(k, wrap(c, v))
		i++
	}
	return sortedMap(hvEntries)
}

// WrapStringToStringMap does not preserve order since order is undefined in a Go map
func WrapStringToStringMap(hash map[string]string) *Hash {
	hvEntries := make([]*HashEntry, len(hash))
	i := 0
	for k, v := range hash {
		hvEntries[i] = WrapHashEntry2(k, stringValue(v))
		i++
	}
	return sortedMap(hvEntries)
}

func sortedMap(hvEntries []*HashEntry) *Hash {
	// map order is undefined (and changes from one run to another) so entries must
	// be sorted to get a predictable order
	sort.Slice(hvEntries, func(i, j int) bool {
		return hvEntries[i].key.String() < hvEntries[j].key.String()
	})
	return &Hash{entries: hvEntries}
}

func WrapStringPValue(hash *hash.StringHash) *Hash {
	hvEntries := make([]*HashEntry, hash.Len())
	i := 0
	hash.EachPair(func(k string, v interface{}) {
		hvEntries[i] = WrapHashEntry2(k, v.(px.Value))
		i++
	})
	return &Hash{entries: hvEntries}
}

func WrapHashFromArray(a *Array) *Hash {
	top := a.Len()
	switch a.PType().(*ArrayType).ElementType().(type) {
	case *ArrayType:
		// Array of arrays. Assume that each nested array is [key, value]
		entries := make([]*HashEntry, top)
		a.EachWithIndex(func(pair px.Value, idx int) {
			pairArr := pair.(px.List)
			if pairArr.Len() != 2 {
				panic(illegalArguments(`Hash`, fmt.Sprintf(`hash entry array must have 2 elements, got %d`, pairArr.Len())))
			}
			entries[idx] = WrapHashEntry(pairArr.At(0), pairArr.At(1))
		})
		return WrapHash(entries)
	default:
		if (top % 2) != 0 {
			panic(illegalArguments(`Hash`, `odd number of arguments in Array`))
		}
		entries := make([]*HashEntry, top/2)
		idx := 0
		a.EachSlice(2, func(slice px.List) {
			entries[idx] = WrapHashEntry(slice.At(0), slice.At(1))
			idx++
		})
		return WrapHash(entries)
	}
}

func IndexedFromArray(a *Array) *Hash {
	top := a.Len()
	entries := make([]*HashEntry, top)
	a.EachWithIndex(func(v px.Value, idx int) {
		entries[idx] = WrapHashEntry(integerValue(int64(idx)), v)
	})
	return WrapHash(entries)
}

func singleMap(key, value px.Value) *Hash {
	return &Hash{entries: []*HashEntry{WrapHashEntry(key, value)}}
}

func singletonMap(key string, value px.Value) px.OrderedMap {
	return &Hash{entries: []*HashEntry{WrapHashEntry2(key, value)}}
}

func (hv *Hash) Add(v px.Value) px.List {
	switch v := v.(type) {
	case *HashEntry:
		return hv.Merge(WrapHash([]*HashEntry{v}))
	case *Array:
		if v.Len() == 2 {
			return hv.Merge(WrapHash([]*HashEntry{WrapHashEntry(v.At(0), v.At(1))}))
		}
	}
	panic(`Operation not supported`)
}

func (hv *Hash) AddAll(v px.List) px.List {
	switch v := v.(type) {
	case *Hash:
		return hv.Merge(v)
	case *Array:
		return hv.Merge(WrapHashFromArray(v))
	}
	panic(`Operation not supported`)
}

func (hv *Hash) All(predicate px.Predicate) bool {
	for _, e := range hv.entries {
		if !predicate(e) {
			return false
		}
	}
	return true
}

func (hv *Hash) AllPairs(predicate px.BiPredicate) bool {
	for _, e := range hv.entries {
		if !predicate(e.key, e.value) {
			return false
		}
	}
	return true
}

func (hv *Hash) AllKeysAreStrings() bool {
	for _, e := range hv.entries {
		if _, ok := e.key.(stringValue); !ok {
			return false
		}
	}
	return true
}

func (hv *Hash) Any(predicate px.Predicate) bool {
	for _, e := range hv.entries {
		if predicate(e) {
			return true
		}
	}
	return false
}

func (hv *Hash) AnyPair(predicate px.BiPredicate) bool {
	for _, e := range hv.entries {
		if predicate(e.key, e.value) {
			return true
		}
	}
	return false
}

func (hv *Hash) AppendEntriesTo(entries []*HashEntry) []*HashEntry {
	return append(entries, hv.entries...)
}

func (hv *Hash) AppendTo(slice []px.Value) []px.Value {
	for _, e := range hv.entries {
		slice = append(slice, e)
	}
	return slice
}

func (hv *Hash) AsArray() px.List {
	values := make([]px.Value, len(hv.entries))
	for idx, entry := range hv.entries {
		values[idx] = WrapValues([]px.Value{entry.key, entry.value})
	}
	return WrapValues(values)
}

func (hv *Hash) At(i int) px.Value {
	if i >= 0 && i < len(hv.entries) {
		return hv.entries[i]
	}
	return undef
}

func (hv *Hash) Delete(key px.Value) px.List {
	if idx, ok := hv.valueIndex()[px.ToKey(key)]; ok {
		return WrapHash(append(hv.entries[:idx], hv.entries[idx+1:]...))
	}
	return hv
}

func (hv *Hash) DeleteAll(keys px.List) px.List {
	entries := hv.entries
	valueIndex := hv.valueIndex()
	keys.Each(func(key px.Value) {
		if idx, ok := valueIndex[px.ToKey(key)]; ok {
			entries = append(hv.entries[:idx], hv.entries[idx+1:]...)
		}
	})
	if len(hv.entries) == len(entries) {
		return hv
	}
	return WrapHash(entries)
}

func (hv *Hash) DetailedType() px.Type {
	return hv.privateDetailedType()
}

func (hv *Hash) ElementType() px.Type {
	return hv.PType().(*HashType).EntryType()
}

func (hv *Hash) Entries() px.List {
	return hv
}

func (hv *Hash) Each(consumer px.Consumer) {
	for _, e := range hv.entries {
		consumer(e)
	}
}

func (hv *Hash) EachSlice(n int, consumer px.SliceConsumer) {
	top := len(hv.entries)
	for i := 0; i < top; i += n {
		e := i + n
		if e > top {
			e = top
		}
		consumer(WrapValues(ValueSlice(hv.entries[i:e])))
	}
}

func (hv *Hash) EachKey(consumer px.Consumer) {
	for _, e := range hv.entries {
		consumer(e.key)
	}
}

func (hv *Hash) Find(predicate px.Predicate) (px.Value, bool) {
	for _, e := range hv.entries {
		if predicate(e) {
			return e, true
		}
	}
	return nil, false
}

func (hv *Hash) Flatten() px.List {
	els := make([]px.Value, 0, len(hv.entries)*2)
	for _, he := range hv.entries {
		els = append(els, he.key, he.value)
	}
	return WrapValues(els).Flatten()
}

func (hv *Hash) Map(mapper px.Mapper) px.List {
	mapped := make([]px.Value, len(hv.entries))
	for i, e := range hv.entries {
		mapped[i] = mapper(e)
	}
	return WrapValues(mapped)
}

func (hv *Hash) MapEntries(mapper px.EntryMapper) px.OrderedMap {
	mapped := make([]*HashEntry, len(hv.entries))
	for i, e := range hv.entries {
		mapped[i] = mapper(e).(*HashEntry)
	}
	return WrapHash(mapped)
}

func (hv *Hash) MapValues(mapper px.Mapper) px.OrderedMap {
	mapped := make([]*HashEntry, len(hv.entries))
	for i, e := range hv.entries {
		mapped[i] = WrapHashEntry(e.key, mapper(e.value))
	}
	return WrapHash(mapped)
}

func (hv *Hash) Select(predicate px.Predicate) px.List {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *Hash) SelectPairs(predicate px.BiPredicate) px.OrderedMap {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if predicate(e.key, e.value) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *Hash) Reflect(c px.Context) reflect.Value {
	ht, ok := ReflectType(c, hv.PType())
	if !ok {
		ht = reflect.TypeOf(map[interface{}]interface{}{})
	}

	keyType := ht.Key()
	valueType := ht.Elem()
	m := reflect.MakeMapWithSize(ht, hv.Len())
	rf := c.Reflector()
	for _, e := range hv.entries {
		m.SetMapIndex(rf.Reflect2(e.key, keyType), rf.Reflect2(e.value, valueType))
	}
	return m
}

func (hv *Hash) ReflectTo(c px.Context, value reflect.Value) {
	ht := value.Type()
	ptr := ht.Kind() == reflect.Ptr
	if ptr {
		ht = ht.Elem()
	}
	if ht.Kind() == reflect.Interface {
		var ok bool
		if ht, ok = ReflectType(c, hv.PType()); !ok {
			ht = reflect.TypeOf(map[interface{}]interface{}{})
		}
	}
	keyType := ht.Key()
	valueType := ht.Elem()
	m := reflect.MakeMapWithSize(ht, hv.Len())
	rf := c.Reflector()
	for _, e := range hv.entries {
		m.SetMapIndex(rf.Reflect2(e.key, keyType), rf.Reflect2(e.value, valueType))
	}
	if ptr {
		// The created map cannot be addressed. A pointer to it is necessary
		x := reflect.New(m.Type())
		x.Elem().Set(m)
		m = x
	}
	value.Set(m)
}

func (hv *Hash) Reduce(redactor px.BiMapper) px.Value {
	if hv.IsEmpty() {
		return undef
	}
	return reduceEntries(hv.entries[1:], hv.At(0), redactor)
}

func (hv *Hash) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduceEntries(hv.entries, initialValue, redactor)
}

func (hv *Hash) Reject(predicate px.Predicate) px.List {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if !predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *Hash) RejectPairs(predicate px.BiPredicate) px.OrderedMap {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if !predicate(e.key, e.value) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *Hash) EachPair(consumer px.BiConsumer) {
	for _, e := range hv.entries {
		consumer(e.key, e.value)
	}
}

func (hv *Hash) EachValue(consumer px.Consumer) {
	for _, e := range hv.entries {
		consumer(e.value)
	}
}

func (hv *Hash) EachWithIndex(consumer px.IndexedConsumer) {
	for i, e := range hv.entries {
		consumer(e, i)
	}
}

func (hv *Hash) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*Hash); ok {
		if top := len(hv.entries); top == len(ov.entries) {
			ovIndex := ov.valueIndex()
			for key, idx := range hv.valueIndex() {
				var ovIdx int
				if ovIdx, ok = ovIndex[key]; !(ok && hv.entries[idx].Equals(ov.entries[ovIdx], g)) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func (hv *Hash) Get(key px.Value) (px.Value, bool) {
	return hv.get(px.ToKey(key))
}

func (hv *Hash) Get2(key px.Value, dflt px.Value) px.Value {
	return hv.get2(px.ToKey(key), dflt)
}

func (hv *Hash) Get3(key px.Value, dflt px.Producer) px.Value {
	return hv.get3(px.ToKey(key), dflt)
}

func (hv *Hash) Get4(key string) (px.Value, bool) {
	return hv.get(px.HashKey(key))
}

func (hv *Hash) Get5(key string, dflt px.Value) px.Value {
	return hv.get2(px.HashKey(key), dflt)
}

func (hv *Hash) Get6(key string, dflt px.Producer) px.Value {
	return hv.get3(px.HashKey(key), dflt)
}

func (hv *Hash) GetEntry(key string) (px.MapEntry, bool) {
	if pos, ok := hv.valueIndex()[px.HashKey(key)]; ok {
		return hv.entries[pos], true
	}
	return nil, false
}

func (hv *Hash) GetEntryFold(key string) (px.MapEntry, bool) {
	for _, e := range hv.entries {
		if strings.EqualFold(e.key.String(), key) {
			return e, true
		}
	}
	return nil, false
}

func (hv *Hash) get(key px.HashKey) (px.Value, bool) {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value, true
	}
	return undef, false
}

func (hv *Hash) get2(key px.HashKey, dflt px.Value) px.Value {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value
	}
	return dflt
}

func (hv *Hash) get3(key px.HashKey, dflt px.Producer) px.Value {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value
	}
	return dflt()
}

func (hv *Hash) IncludesKey(o px.Value) bool {
	_, ok := hv.valueIndex()[px.ToKey(o)]
	return ok
}

func (hv *Hash) IncludesKey2(key string) bool {
	_, ok := hv.valueIndex()[px.HashKey(key)]
	return ok
}

func (hv *Hash) IsEmpty() bool {
	return len(hv.entries) == 0
}

func (hv *Hash) IsHashStyle() bool {
	return true
}

func (hv *Hash) Keys() px.List {
	keys := make([]px.Value, len(hv.entries))
	for idx, entry := range hv.entries {
		keys[idx] = entry.key
	}
	return WrapValues(keys)
}

func (hv *Hash) Len() int {
	return len(hv.entries)
}

func (hv *Hash) Merge(o px.OrderedMap) px.OrderedMap {
	return WrapHash(hv.mergeEntries(o))
}

func (hv *Hash) mergeEntries(o px.OrderedMap) []*HashEntry {
	oh := o.(*Hash)
	index := hv.valueIndex()
	selfLen := len(hv.entries)
	all := make([]*HashEntry, selfLen, selfLen+len(oh.entries))
	copy(all, hv.entries)
	for _, entry := range oh.entries {
		if idx, ok := index[px.ToKey(entry.key)]; ok {
			all[idx] = entry
		} else {
			all = append(all, entry)
		}
	}
	return all
}

func (hv *Hash) Slice(i int, j int) px.List {
	return WrapHash(hv.entries[i:j])
}

type hashSorter struct {
	entries    []*HashEntry
	comparator px.Comparator
}

func (s *hashSorter) Len() int {
	return len(s.entries)
}

func (s *hashSorter) Less(i, j int) bool {
	vs := s.entries
	return s.comparator(vs[i].key, vs[j].key)
}

func (s *hashSorter) Swap(i, j int) {
	vs := s.entries
	v := vs[i]
	vs[i] = vs[j]
	vs[j] = v
}

// Sort reorders the associations of this hash by applying the comparator
// to the keys
func (hv *Hash) Sort(comparator px.Comparator) px.List {
	s := &hashSorter{make([]*HashEntry, len(hv.entries)), comparator}
	copy(s.entries, hv.entries)
	sort.Sort(s)
	return WrapHash(s.entries)
}

func (hv *Hash) String() string {
	return px.ToString2(hv, None)
}

func (hv *Hash) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	hv.ToString2(b, s, px.GetFormat(s.FormatMap(), hv.PType()), '{', g)
}

func (hv *Hash) ToString2(b io.Writer, s px.FormatContext, f px.Format, delim byte, g px.RDetect) {
	if g == nil {
		g = make(px.RDetect)
	} else if g[hv] {
		utils.WriteString(b, `<recursive reference>`)
		return
	}
	g[hv] = true

	switch f.FormatChar() {
	case 'a':
		WrapArray3(hv).ToString(b, s, g)
		return
	case 'h', 's', 'p':
		indent := s.Indentation()
		indent = indent.Indenting(f.IsAlt() || indent.IsIndenting())

		if indent.Breaks() && delim != '(' {
			utils.WriteString(b, "\n")
			utils.WriteString(b, indent.Padding())
		}

		var delims [2]byte
		if delim == '(' || f.LeftDelimiter() == 0 {
			delims = delimiterPairs[delim]
		} else {
			delims = delimiterPairs[f.LeftDelimiter()]
		}
		if delims[0] != 0 {
			utils.WriteByte(b, delims[0])
		}

		if f.IsAlt() {
			utils.WriteString(b, "\n")
		}

		top := len(hv.entries)
		if top > 0 {
			sep := f.Separator(`,`)
			assoc := f.Separator2(` => `)
			cf := f.ContainerFormats()
			if cf == nil {
				cf = DefaultContainerFormats
			}
			if f.IsAlt() {
				sep += "\n"
			} else {
				sep += ` `
			}

			childrenIndent := indent.Increase(f.IsAlt())
			padding := ``
			if f.IsAlt() {
				padding = childrenIndent.Padding()
			}

			last := top - 1
			for idx, entry := range hv.entries {
				k := entry.Key()
				utils.WriteString(b, padding)
				if isContainer(k, s) {
					k.ToString(b, px.NewFormatContext2(childrenIndent, s.FormatMap(), s.Properties()), g)
				} else {
					k.ToString(b, px.NewFormatContext2(childrenIndent, cf, s.Properties()), g)
				}
				v := entry.Value()
				utils.WriteString(b, assoc)
				if isContainer(v, s) {
					v.ToString(b, px.NewFormatContext2(childrenIndent, s.FormatMap(), s.Properties()), g)
				} else {
					if v == nil {
						panic(`not good`)
					}
					v.ToString(b, px.NewFormatContext2(childrenIndent, cf, s.Properties()), g)
				}
				if idx < last {
					utils.WriteString(b, sep)
				}
			}
		}

		if f.IsAlt() {
			utils.WriteString(b, "\n")
			utils.WriteString(b, indent.Padding())
		}
		if delims[1] != 0 {
			utils.WriteByte(b, delims[1])
		}
	default:
		panic(s.UnsupportedFormat(hv.PType(), `hasp`, f))
	}
	delete(g, hv)
}

func (hv *Hash) PType() px.Type {
	return hv.privateReducedType()
}

// Unique on a Hash will always return self since the keys of a hash are unique
func (hv *Hash) Unique() px.List {
	return hv
}

func (hv *Hash) Values() px.List {
	values := make([]px.Value, len(hv.entries))
	for idx, entry := range hv.entries {
		values[idx] = entry.value
	}
	return WrapValues(values)
}

func (hv *Hash) privateDetailedType() px.Type {
	if hv.detailedType == nil {
		top := len(hv.entries)
		if top == 0 {
			hv.detailedType = hv.privateReducedType()
			return hv.detailedType
		}

		structEntries := make([]*StructElement, top)
		for idx, entry := range hv.entries {
			if ks, ok := entry.key.(stringValue); ok {
				structEntries[idx] = NewStructElement(ks, DefaultAnyType())
				continue
			}

			// Struct type cannot be used unless all keys are strings
			hv.detailedType = hv.privateReducedType()
			return hv.detailedType
		}
		hv.detailedType = NewStructType(structEntries)

		for _, entry := range hv.entries {
			if sv, ok := entry.key.(stringValue); !ok || len(string(sv)) == 0 {
				firstEntry := hv.entries[0]
				commonKeyType := px.DetailedValueType(firstEntry.key)
				commonValueType := px.DetailedValueType(firstEntry.value)
				for idx := 1; idx < top; idx++ {
					entry := hv.entries[idx]
					commonKeyType = commonType(commonKeyType, px.DetailedValueType(entry.key))
					commonValueType = commonType(commonValueType, px.DetailedValueType(entry.value))
				}
				sz := int64(len(hv.entries))
				hv.detailedType = NewHashType(commonKeyType, commonValueType, NewIntegerType(sz, sz))
				return hv.detailedType
			}
		}

		for idx, entry := range hv.entries {
			structEntries[idx] = NewStructElement(entry.key, px.DetailedValueType(entry.value))
		}
	}
	return hv.detailedType
}

func (hv *Hash) privateReducedType() px.Type {
	if hv.reducedType == nil {
		top := len(hv.entries)
		if top == 0 {
			hv.reducedType = EmptyHashType()
		} else {
			sz := int64(top)
			ht := NewHashType(DefaultAnyType(), DefaultAnyType(), NewIntegerType(sz, sz))
			hv.reducedType = ht
			firstEntry := hv.entries[0]
			commonKeyType := firstEntry.key.PType()
			commonValueType := firstEntry.value.PType()
			for idx := 1; idx < top; idx++ {
				entry := hv.entries[idx]
				commonKeyType = commonType(commonKeyType, entry.key.PType())
				commonValueType = commonType(commonValueType, entry.value.PType())
			}
			ht.keyType = commonKeyType
			ht.valueType = commonValueType
		}
	}
	return hv.reducedType
}

func (hv *Hash) valueIndex() map[px.HashKey]int {
	if hv.index == nil {
		result := make(map[px.HashKey]int, len(hv.entries))
		for idx, entry := range hv.entries {
			result[px.ToKey(entry.key)] = idx
		}
		hv.index = result
	}
	return hv.index
}

func NewMutableHash() *MutableHashValue {
	return &MutableHashValue{Hash{entries: make([]*HashEntry, 0, 7)}}
}

// PutAll merges the given hash into this hash (mutates the hash). The method
// is not thread safe
func (hv *MutableHashValue) PutAll(o px.OrderedMap) {
	hv.entries = hv.mergeEntries(o)
	hv.detailedType = nil
	hv.index = nil
}

// Put adds or replaces the given key/value association in this hash
func (hv *MutableHashValue) Put(key, value px.Value) {
	hv.PutAll(WrapHash([]*HashEntry{{key, value}}))
}

func reduceEntries(slice []*HashEntry, initialValue px.Value, redactor px.BiMapper) px.Value {
	memo := initialValue
	for _, v := range slice {
		memo = redactor(memo, v)
	}
	return memo
}
