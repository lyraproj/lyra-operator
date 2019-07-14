package types

import (
	"bytes"
	"io"
	"math"
	"reflect"
	"sort"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type (
	ArrayType struct {
		size *IntegerType
		typ  px.Type
	}

	Array struct {
		reducedType  *ArrayType
		detailedType px.Type
		elements     []px.Value
	}
)

var ArrayMetaType px.ObjectType

func init() {
	ArrayMetaType = newObjectType(`Pcore::ArrayType`,
		`Pcore::CollectionType {
  attributes => {
    'element_type' => { type => Type, value => Any }
  },
  serialization => [ 'element_type', 'size_type' ]
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newArrayType2(args...)
		},
		func(ctx px.Context, args []px.Value) px.Value {
			h := args[0].(*Hash)
			et := h.Get5(`element_type`, DefaultAnyType())
			st := h.Get5(`size_type`, PositiveIntegerType())
			return newArrayType2(et, st)
		})

	newGoConstructor3([]string{`Array`, `Tuple`}, nil,
		func(d px.Dispatch) {
			d.Param(`Variant[Array,Hash,Binary,Iterable]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				switch arg := args[0].(type) {
				case *Array:
					if len(args) > 1 && args[1].(booleanValue).Bool() {
						// Wrapped
						return WrapValues(args[:1])
					}
					return arg
				default:
					return arg.(px.Arrayable).AsArray()
				}
			})
		},
	)
}

func DefaultArrayType() *ArrayType {
	return arrayTypeDefault
}

func EmptyArrayType() *ArrayType {
	return arrayTypeEmpty
}

func NewArrayType(element px.Type, rng *IntegerType) *ArrayType {
	if element == nil {
		element = anyTypeDefault
	}
	if rng == nil {
		rng = IntegerTypePositive
	}
	if *rng == *IntegerTypePositive && element == anyTypeDefault {
		return DefaultArrayType()
	}
	if *rng == *IntegerTypeZero && element == unitTypeDefault {
		return EmptyArrayType()
	}
	return &ArrayType{rng, element}
}

func newArrayType2(args ...px.Value) *ArrayType {
	argc := len(args)
	if argc == 0 {
		return DefaultArrayType()
	}

	offset := 0
	element, ok := args[0].(px.Type)
	if ok {
		offset++
	} else {
		element = DefaultAnyType()
	}

	var rng *IntegerType
	switch argc - offset {
	case 0:
		rng = IntegerTypePositive
	case 1:
		sizeArg := args[offset]
		if rng, ok = sizeArg.(*IntegerType); !ok {
			var sz int64
			sz, ok = toInt(sizeArg)
			if !ok {
				panic(illegalArgumentType(`Array[]`, offset, `Variant[Integer, Type[Integer]]`, sizeArg))
			}
			rng = NewIntegerType(sz, math.MaxInt64)
		}
	case 2:
		var min, max int64
		arg := args[offset]
		if min, ok = toInt(arg); !ok {
			if _, ok = arg.(*DefaultValue); !ok {
				panic(illegalArgumentType(`Array[]`, offset, `Integer`, arg))
			}
			min = 0
		}
		offset++
		arg = args[offset]
		if max, ok = toInt(args[offset]); !ok {
			if _, ok = arg.(*DefaultValue); !ok {
				panic(illegalArgumentType(`Array[]`, offset, `Integer`, arg))
			}
			max = math.MaxInt64
		}
		rng = NewIntegerType(min, max)
	default:
		panic(illegalArgumentCount(`Array[]`, `0 - 3`, argc))
	}
	return NewArrayType(element, rng)
}

func (t *ArrayType) ElementType() px.Type {
	return t.typ
}

func (t *ArrayType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.size.Accept(v, g)
	t.typ.Accept(v, g)
}

func (t *ArrayType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*ArrayType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *ArrayType) Generic() px.Type {
	if t.typ == anyTypeDefault {
		return arrayTypeDefault
	}
	return NewArrayType(px.Generalize(t.typ), nil)
}

func (t *ArrayType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `element_type`:
		return t.typ, true
	case `size_type`:
		return t.size, true
	}
	return nil, false
}

func (t *ArrayType) Default() px.Type {
	return arrayTypeDefault
}

func (t *ArrayType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o := o.(type) {
	case *ArrayType:
		return t.size.IsAssignable(o.size, g) && GuardedIsAssignable(t.typ, o.typ, g)
	case *TupleType:
		return t.size.IsAssignable(o.givenOrActualSize, g) && allAssignableTo(o.types, t.typ, g)
	default:
		return false
	}
}

func (t *ArrayType) IsInstance(v px.Value, g px.Guard) bool {
	iv, ok := v.(*Array)
	if !ok {
		return false
	}

	osz := iv.Len()
	if !t.size.IsInstance3(osz) {
		return false
	}

	if t.typ == anyTypeDefault {
		return true
	}

	for idx := 0; idx < osz; idx++ {
		if !GuardedIsInstance(t.typ, iv.At(idx), g) {
			return false
		}
	}
	return true
}

func (t *ArrayType) MetaType() px.ObjectType {
	return ArrayMetaType
}

func (t *ArrayType) Name() string {
	return `Array`
}

func (t *ArrayType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *ArrayType) Size() *IntegerType {
	return t.size
}

func (t *ArrayType) String() string {
	return px.ToString2(t, None)
}

func (t *ArrayType) PType() px.Type {
	return &TypeType{t}
}

func (t *ArrayType) Parameters() []px.Value {
	if t.typ.Equals(unitTypeDefault, nil) && *t.size == *IntegerTypeZero {
		return t.size.SizeParameters()
	}

	params := make([]px.Value, 0)
	if !t.typ.Equals(DefaultAnyType(), nil) {
		params = append(params, t.typ)
	}
	if *t.size != *IntegerTypePositive {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *ArrayType) ReflectType(c px.Context) (reflect.Type, bool) {
	if et, ok := ReflectType(c, t.ElementType()); ok {
		return reflect.SliceOf(et), true
	}
	return nil, false
}

func (t *ArrayType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *ArrayType) SerializationString() string {
	return t.String()
}

func (t *ArrayType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

var arrayTypeDefault = &ArrayType{IntegerTypePositive, anyTypeDefault}
var arrayTypeEmpty = &ArrayType{IntegerTypeZero, unitTypeDefault}

func BuildArray(len int, bld func(*Array, []px.Value) []px.Value) *Array {
	ar := &Array{elements: make([]px.Value, 0, len)}
	ar.elements = bld(ar, ar.elements)
	return ar
}

func SingletonArray(element px.Value) *Array {
	return &Array{elements: []px.Value{element}}
}

func WrapTypes(elements []px.Type) *Array {
	els := make([]px.Value, len(elements))
	for i, e := range elements {
		if e == nil {
			panic(px.Error(px.NilArrayElement, issue.H{`index`: i}))
		}
		els[i] = e
	}
	return &Array{elements: els}
}

func WrapValues(elements []px.Value) *Array {
	for i, e := range elements {
		if e == nil {
			panic(px.Error(px.NilArrayElement, issue.H{`index`: i}))
		}
	}
	return &Array{elements: elements}
}

func WrapInterfaces(c px.Context, elements []interface{}) *Array {
	els := make([]px.Value, len(elements))
	for i, e := range elements {
		els[i] = wrap(c, e)
	}
	return &Array{elements: els}
}

func WrapInts(ints []int) *Array {
	els := make([]px.Value, len(ints))
	for i, e := range ints {
		els[i] = integerValue(int64(e))
	}
	return &Array{elements: els}
}

func WrapStrings(strings []string) *Array {
	els := make([]px.Value, len(strings))
	for i, e := range strings {
		els[i] = stringValue(e)
	}
	return &Array{elements: els}
}

func WrapArray3(iv px.List) *Array {
	if ar, ok := iv.(*Array); ok {
		return ar
	}
	return WrapValues(iv.AppendTo(make([]px.Value, 0, iv.Len())))
}

func (av *Array) Add(ov px.Value) px.List {
	return WrapValues(append(av.elements, ov))
}

func (av *Array) AddAll(ov px.List) px.List {
	if ar, ok := ov.(*Array); ok {
		return WrapValues(append(av.elements, ar.elements...))
	}

	aLen := len(av.elements)
	sLen := aLen + ov.Len()
	el := make([]px.Value, sLen)
	copy(el, av.elements)
	for idx := aLen; idx < sLen; idx++ {
		el[idx] = ov.At(idx - aLen)
	}
	return WrapValues(el)
}

func (av *Array) All(predicate px.Predicate) bool {
	return px.All(av.elements, predicate)
}

func (av *Array) Any(predicate px.Predicate) bool {
	return px.Any(av.elements, predicate)
}

func (av *Array) AppendTo(slice []px.Value) []px.Value {
	return append(slice, av.elements...)
}

func (av *Array) At(i int) px.Value {
	if i >= 0 && i < len(av.elements) {
		return av.elements[i]
	}
	return undef
}

func (av *Array) Delete(ov px.Value) px.List {
	return av.Reject(func(elem px.Value) bool {
		return elem.Equals(ov, nil)
	})
}

func (av *Array) DeleteAll(ov px.List) px.List {
	return av.Reject(func(elem px.Value) bool {
		return ov.Any(func(oe px.Value) bool {
			return elem.Equals(oe, nil)
		})
	})
}

func (av *Array) DetailedType() px.Type {
	return av.privateDetailedType()
}

func (av *Array) Each(consumer px.Consumer) {
	for _, e := range av.elements {
		consumer(e)
	}
}

func (av *Array) EachWithIndex(consumer px.IndexedConsumer) {
	for i, e := range av.elements {
		consumer(e, i)
	}
}

func (av *Array) EachSlice(n int, consumer px.SliceConsumer) {
	top := len(av.elements)
	for i := 0; i < top; i += n {
		e := i + n
		if e > top {
			e = top
		}
		consumer(WrapValues(av.elements[i:e]))
	}
}

func (av *Array) ElementType() px.Type {
	return av.PType().(*ArrayType).ElementType()
}

func (av *Array) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*Array); ok {
		if top := len(av.elements); top == len(ov.elements) {
			for idx := 0; idx < top; idx++ {
				if !av.elements[idx].Equals(ov.elements[idx], g) {
					return false
				}
			}
			return true
		}
	}
	if len(av.elements) == 2 {
		if he, ok := o.(*HashEntry); ok {
			return av.elements[0].Equals(he.key, g) && av.elements[1].Equals(he.value, g)
		}
	}
	return false
}

func (av *Array) Find(predicate px.Predicate) (px.Value, bool) {
	return px.Find(av.elements, predicate)
}

func (av *Array) Flatten() px.List {
	for _, e := range av.elements {
		switch e.(type) {
		case *Array, *HashEntry:
			return WrapValues(flattenElements(av.elements, make([]px.Value, 0, len(av.elements)*2)))
		}
	}
	return av
}

func flattenElements(elements, receiver []px.Value) []px.Value {
	for _, e := range elements {
		switch e := e.(type) {
		case *Array:
			receiver = flattenElements(e.elements, receiver)
		case *HashEntry:
			receiver = flattenElements([]px.Value{e.key, e.value}, receiver)
		default:
			receiver = append(receiver, e)
		}
	}
	return receiver
}

func (av *Array) IsEmpty() bool {
	return len(av.elements) == 0
}

func (av *Array) IsHashStyle() bool {
	return false
}

func (av *Array) Len() int {
	return len(av.elements)
}

func (av *Array) Map(mapper px.Mapper) px.List {
	return WrapValues(px.Map(av.elements, mapper))
}

func (av *Array) Reduce(redactor px.BiMapper) px.Value {
	if av.IsEmpty() {
		return undef
	}
	return reduceSlice(av.elements[1:], av.At(0), redactor)
}

func (av *Array) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduceSlice(av.elements, initialValue, redactor)
}

func (av *Array) Reflect(c px.Context) reflect.Value {
	at, ok := ReflectType(c, av.PType())
	if !ok {
		at = reflect.TypeOf([]interface{}{})
	}
	s := reflect.MakeSlice(at, av.Len(), av.Len())
	rf := c.Reflector()
	for i, e := range av.elements {
		rf.ReflectTo(e, s.Index(i))
	}
	return s
}

func (av *Array) ReflectTo(c px.Context, value reflect.Value) {
	vt := value.Type()
	ptr := vt.Kind() == reflect.Ptr
	if ptr {
		vt = vt.Elem()
	}
	s := reflect.MakeSlice(vt, av.Len(), av.Len())
	rf := c.Reflector()
	for i, e := range av.elements {
		rf.ReflectTo(e, s.Index(i))
	}
	if ptr {
		// The created slice cannot be addressed. A pointer to it is necessary
		x := reflect.New(s.Type())
		x.Elem().Set(s)
		s = x
	}
	value.Set(s)
}

func (av *Array) Reject(predicate px.Predicate) px.List {
	return WrapValues(px.Reject(av.elements, predicate))
}

func (av *Array) Select(predicate px.Predicate) px.List {
	return WrapValues(px.Select(av.elements, predicate))
}

func (av *Array) Slice(i int, j int) px.List {
	return WrapValues(av.elements[i:j])
}

type arraySorter struct {
	values     []px.Value
	comparator px.Comparator
}

func (s *arraySorter) Len() int {
	return len(s.values)
}

func (s *arraySorter) Less(i, j int) bool {
	vs := s.values
	return s.comparator(vs[i], vs[j])
}

func (s *arraySorter) Swap(i, j int) {
	vs := s.values
	v := vs[i]
	vs[i] = vs[j]
	vs[j] = v
}

func (av *Array) Sort(comparator px.Comparator) px.List {
	s := &arraySorter{make([]px.Value, len(av.elements)), comparator}
	copy(s.values, av.elements)
	sort.Sort(s)
	return WrapValues(s.values)
}

func (av *Array) String() string {
	return px.ToString2(av, None)
}

func (av *Array) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	av.ToString2(b, s, px.GetFormat(s.FormatMap(), av.PType()), '[', g)
}

func (av *Array) ToString2(b io.Writer, s px.FormatContext, f px.Format, delim byte, g px.RDetect) {
	if g == nil {
		g = make(px.RDetect)
	} else if g[av] {
		utils.WriteString(b, `<recursive reference>`)
		return
	}
	g[av] = true

	switch f.FormatChar() {
	case 'a', 's', 'p':
	default:
		panic(s.UnsupportedFormat(av.PType(), `asp`, f))
	}
	indent := s.Indentation()
	indent = indent.Indenting(f.IsAlt() || indent.IsIndenting())

	if indent.Breaks() {
		utils.WriteString(b, "\n")
		utils.WriteString(b, indent.Padding())
	}

	var delims [2]byte
	if f.LeftDelimiter() == 0 {
		delims = delimiterPairs[delim]
	} else {
		delims = delimiterPairs[f.LeftDelimiter()]
	}
	if delims[0] != 0 {
		utils.WriteByte(b, delims[0])
	}

	top := len(av.elements)
	if top > 0 {
		mapped := make([]string, top)
		arrayOrHash := make([]bool, top)
		childrenIndent := indent.Increase(f.IsAlt())

		cf := f.ContainerFormats()
		if cf == nil {
			cf = DefaultContainerFormats
		}
		for idx, v := range av.elements {
			arrayOrHash[idx] = isContainer(v, s)
			mapped[idx] = childToString(v, childrenIndent.Subsequent(), s, cf, g)
		}

		szBreak := false
		if f.IsAlt() && f.Width() >= 0 {
			widest := 0
			for idx, ah := range arrayOrHash {
				if ah {
					widest = 0
				} else {
					widest += len(mapped[idx])
					if widest > f.Width() {
						szBreak = true
						break
					}
				}
			}
		}

		sep := f.Separator(`,`)
		for idx, ah := range arrayOrHash {
			if childrenIndent.IsFirst() {
				childrenIndent = childrenIndent.Subsequent()
				// if breaking, indent first element by one
				if szBreak && !ah {
					utils.WriteString(b, ` `)
				}
			} else {
				utils.WriteString(b, sep)
				// if break on each (and breaking will not occur because next is an array or hash)
				// or, if indenting, and previous was an array or hash, then break and continue on next line
				// indented.
				if !ah && (szBreak || f.IsAlt() && arrayOrHash[idx-1]) {
					utils.WriteString(b, "\n")
					utils.WriteString(b, childrenIndent.Padding())
				} else if !(f.IsAlt() && ah) {
					utils.WriteString(b, ` `)
				}
			}
			utils.WriteString(b, mapped[idx])
		}
	}
	if delims[1] != 0 {
		utils.WriteByte(b, delims[1])
	}
	delete(g, av)
}

func (av *Array) Unique() px.List {
	top := len(av.elements)
	if top < 2 {
		return av
	}

	result := make([]px.Value, 0, top)
	exists := make(map[px.HashKey]bool, top)
	for _, v := range av.elements {
		key := px.ToKey(v)
		if !exists[key] {
			exists[key] = true
			result = append(result, v)
		}
	}
	if len(result) == len(av.elements) {
		return av
	}
	return WrapValues(result)
}

func childToString(child px.Value, indent px.Indentation, parentCtx px.FormatContext, cf px.FormatMap, g px.RDetect) string {
	var childrenCtx px.FormatContext
	if isContainer(child, parentCtx) {
		childrenCtx = newFormatContext2(indent, parentCtx.FormatMap(), parentCtx.Properties())
	} else {
		childrenCtx = newFormatContext2(indent, cf, parentCtx.Properties())
	}
	b := bytes.NewBufferString(``)
	child.ToString(b, childrenCtx, g)
	return b.String()
}

func isContainer(child px.Value, s px.FormatContext) bool {
	switch child.(type) {
	case *Array, *Hash:
		return true
	case px.ObjectType, px.TypeSet:
		if ex, ok := s.Property(`expanded`); ok && ex == `true` {
			return true
		}
		return false
	case px.PuppetObject:
		return true
	default:
		return false
	}
}

func (av *Array) PType() px.Type {
	return av.privateReducedType()
}

func (av *Array) privateDetailedType() px.Type {
	if av.detailedType == nil {
		if len(av.elements) == 0 {
			av.detailedType = av.privateReducedType()
		} else {
			types := make([]px.Type, len(av.elements))
			av.detailedType = NewTupleType(types, nil)
			for idx := range types {
				types[idx] = DefaultAnyType()
			}
			for idx, element := range av.elements {
				types[idx] = px.DetailedValueType(element)
			}
		}
	}
	return av.detailedType
}

func (av *Array) privateReducedType() *ArrayType {
	if av.reducedType == nil {
		top := len(av.elements)
		if top == 0 {
			av.reducedType = EmptyArrayType()
		} else {
			av.reducedType = NewArrayType(DefaultAnyType(), NewIntegerType(int64(top), int64(top)))
			elemType := av.elements[0].PType()
			for idx := 1; idx < top; idx++ {
				elemType = commonType(elemType, av.elements[idx].PType())
			}
			av.reducedType.typ = elemType
		}
	}
	return av.reducedType
}

func reduceSlice(slice []px.Value, initialValue px.Value, redactor px.BiMapper) px.Value {
	memo := initialValue
	for _, v := range slice {
		memo = redactor(memo, v)
	}
	return memo
}
