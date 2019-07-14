package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"reflect"
	"regexp"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type (
	// String that is unconstrained
	stringType struct{}

	// String constrained to content
	vcStringType struct {
		stringType
		value string
	}

	// String constrained by length of string
	scStringType struct {
		stringType
		size *IntegerType
	}

	// stringValue represents string as a pcore.Value
	stringValue string
)

var stringTypeDefault = &stringType{}
var stringTypeNotEmpty = &scStringType{size: NewIntegerType(1, math.MaxInt64)}

var StringMetaType px.ObjectType

func init() {
	StringMetaType = newObjectType(`Pcore::StringType`, `Pcore::ScalarDataType {
	attributes => {
		size_type_or_value => {
			type => Variant[Undef,String,Type[Integer]],
			value => undef
		},
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
		return newStringType2(args...)
	})

	newGoConstructor2(`String`,
		func(t px.LocalTypes) {
			t.Type2(`Format`, NewPatternType([]*RegexpType{NewRegexpTypeR(px.FormatPattern)}))
			t.Type(`ContainerFormat`, `Struct[{
          Optional[format]         => Format,
          Optional[separator]      => String,
          Optional[separator2]     => String,
          Optional[string_formats] => Hash[Type, Format]
        }]`)
			t.Type(`TypeMap`, `Hash[Type, Variant[Format, ContainerFormat]]`)
			t.Type(`Formats`, `Variant[Default, String[1], TypeMap]`)
		},

		func(d px.Dispatch) {
			d.Param(`Any`)
			d.OptionalParam(`Formats`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				f := None
				if len(args) > 1 {
					var err error
					f, err = px.NewFormatContext3(args[0], args[1])
					if err != nil {
						panic(illegalArgument(`String`, 1, err.Error()))
					}
				}
				return stringValue(px.ToString2(args[0], f))
			})
		},
	)
}

func DefaultStringType() *stringType {
	return stringTypeDefault
}

func NewStringType(rng *IntegerType, s string) px.Type {
	if s == `` {
		if rng == nil || *rng == *IntegerTypePositive {
			return DefaultStringType()
		}
		return &scStringType{size: rng}
	}
	return &vcStringType{value: s}
}

func newStringType2(args ...px.Value) px.Type {
	var rng *IntegerType
	var ok bool
	switch len(args) {
	case 0:
		return DefaultStringType()
	case 1:
		var value stringValue
		if value, ok = args[0].(stringValue); ok {
			return NewStringType(nil, string(value))
		}
		rng, ok = args[0].(*IntegerType)
		if !ok {
			var min int64
			min, ok = toInt(args[0])
			if !ok {
				panic(illegalArgumentType(`String[]`, 0, `String, Integer or Type[Integer]`, args[0]))
			}
			rng = NewIntegerType(min, math.MaxInt64)
		}
	case 2:
		var min, max int64
		min, ok = toInt(args[0])
		if !ok {
			panic(illegalArgumentType(`String[]`, 0, `Integer`, args[0]))
		}
		max, ok = toInt(args[1])
		if !ok {
			panic(illegalArgumentType(`String[]`, 1, `Integer`, args[1]))
		}
		rng = NewIntegerType(min, max)
	default:
		panic(illegalArgumentCount(`String[]`, `0 - 2`, len(args)))
	}
	return NewStringType(rng, ``)
}

func (t *stringType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *scStringType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.size.Accept(v, g)
}

func (t *stringType) Default() px.Type {
	return stringTypeDefault
}

func (t *stringType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*stringType)
	return ok
}

func (t *scStringType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*scStringType); ok {
		return t.size.Equals(ot.size, g)
	}
	return false
}

func (t *vcStringType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*vcStringType); ok {
		return t.value == ot.value
	}
	return false
}

func (t *stringType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `size_type_or_value`:
		return IntegerTypePositive, true
	}
	return nil, false
}

func (t *scStringType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `size_type_or_value`:
		return t.size, true
	}
	return nil, false
}

func (t *vcStringType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `size_type_or_value`:
		return stringValue(t.value), true
	}
	return nil, false
}

func (t *stringType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o.(type) {
	case *stringType, *scStringType, *vcStringType, *EnumType, *PatternType:
		return true
	}
	return false
}

func (t *scStringType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o := o.(type) {
	case *vcStringType:
		return t.size.IsInstance3(len(o.value))
	case *scStringType:
		return t.size.IsAssignable(o.size, g)
	case *EnumType:
		for _, str := range o.values {
			if !t.size.IsInstance3(len(string(str))) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *vcStringType) IsAssignable(o px.Type, g px.Guard) bool {
	if st, ok := o.(*vcStringType); ok {
		return t.value == st.value
	}
	return false
}

func (t *stringType) IsInstance(o px.Value, g px.Guard) bool {
	_, ok := o.(stringValue)
	return ok
}

func (t *scStringType) IsInstance(o px.Value, g px.Guard) bool {
	str, ok := o.(stringValue)
	return ok && t.size.IsInstance3(len(string(str)))
}

func (t *vcStringType) IsInstance(o px.Value, g px.Guard) bool {
	str, ok := o.(stringValue)
	return ok && t.value == string(str)
}

func (t *stringType) MetaType() px.ObjectType {
	return StringMetaType
}

func (t *stringType) Name() string {
	return `String`
}

func (t *stringType) Parameters() []px.Value {
	return px.EmptyValues
}

func (t *scStringType) Parameters() []px.Value {
	return t.size.Parameters()
}

func (t *stringType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(`x`), true
}

func (t *stringType) CanSerializeAsString() bool {
	return true
}

func (t *stringType) SerializationString() string {
	return t.String()
}

func (t *stringType) String() string {
	return px.ToString2(t, None)
}

func (t *scStringType) String() string {
	return px.ToString2(t, None)
}

func (t *vcStringType) String() string {
	return px.ToString2(t, None)
}

func (t *stringType) Size() px.Type {
	return IntegerTypePositive
}

func (t *scStringType) Size() px.Type {
	return t.size
}

func (t *stringType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *scStringType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *vcStringType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *stringType) PType() px.Type {
	return &TypeType{t}
}

func (t *stringType) Value() *string {
	return nil
}

func (t *vcStringType) Value() *string {
	return &t.value
}

func WrapString(str string) px.StringValue {
	return stringValue(str)
}

func (sv stringValue) Add(v px.Value) px.List {
	if ov, ok := v.(stringValue); ok {
		return stringValue(string(sv) + string(ov))
	}
	panic(fmt.Sprintf(`No auto conversion from %s to String`, v.PType().String()))
}

var OneCharStringType = NewStringType(NewIntegerType(1, 1), ``)

func (sv stringValue) AddAll(tv px.List) px.List {
	s := bytes.NewBufferString(sv.String())
	tv.Each(func(e px.Value) {
		ev, ok := e.(stringValue)
		if !ok {
			panic(fmt.Sprintf(`No auto conversion from %s to String`, e.PType().String()))
		}
		s.WriteString(string(ev))
	})
	return stringValue(s.String())
}

func (sv stringValue) All(predicate px.Predicate) bool {
	for _, c := range sv.String() {
		if !predicate(stringValue(string(c))) {
			return false
		}
	}
	return true
}

func (sv stringValue) Any(predicate px.Predicate) bool {
	for _, c := range sv.String() {
		if predicate(stringValue(string(c))) {
			return true
		}
	}
	return false
}

func (sv stringValue) AppendTo(slice []px.Value) []px.Value {
	for _, c := range sv.String() {
		slice = append(slice, stringValue(string(c)))
	}
	return slice
}

func (sv stringValue) AsArray() px.List {
	return WrapValues(sv.Elements())
}

func (sv stringValue) At(i int) px.Value {
	if i >= 0 && i < len(sv.String()) {
		return stringValue(sv.String()[i : i+1])
	}
	return undef
}

func (sv stringValue) Delete(v px.Value) px.List {
	panic(`Operation not supported`)
}

func (sv stringValue) DeleteAll(tv px.List) px.List {
	panic(`Operation not supported`)
}

func (sv stringValue) Elements() []px.Value {
	str := sv.String()
	top := len(str)
	el := make([]px.Value, top)
	for idx, c := range str {
		el[idx] = stringValue(string(c))
	}
	return el
}

func (sv stringValue) Each(consumer px.Consumer) {
	for _, c := range sv.String() {
		consumer(stringValue(string(c)))
	}
}

func (sv stringValue) EachSlice(n int, consumer px.SliceConsumer) {
	s := sv.String()
	top := len(s)
	for i := 0; i < top; i += n {
		e := i + n
		if e > top {
			e = top
		}
		consumer(stringValue(s[i:e]))
	}
}

func (sv stringValue) EachWithIndex(consumer px.IndexedConsumer) {
	for i, c := range sv.String() {
		consumer(stringValue(string(c)), i)
	}
}

func (sv stringValue) ElementType() px.Type {
	return OneCharStringType
}

func (sv stringValue) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(stringValue); ok {
		return string(sv) == string(ov)
	}
	return false
}

func (sv stringValue) EqualsIgnoreCase(o px.Value) bool {
	if os, ok := o.(stringValue); ok {
		return strings.EqualFold(string(sv), string(os))
	}
	return false
}

func (sv stringValue) Find(predicate px.Predicate) (px.Value, bool) {
	for _, c := range string(sv) {
		e := stringValue(string(c))
		if predicate(e) {
			return e, true
		}
	}
	return nil, false
}

func (sv stringValue) Flatten() px.List {
	return sv
}

func (sv stringValue) IsEmpty() bool {
	return sv.Len() == 0
}

func (sv stringValue) IsHashStyle() bool {
	return false
}

func (sv stringValue) Len() int {
	return len(sv)
}

func (sv stringValue) Map(mapper px.Mapper) px.List {
	s := sv.String()
	mapped := make([]px.Value, len(s))
	for i, c := range s {
		mapped[i] = mapper(stringValue(string(c)))
	}
	return WrapValues(mapped)
}

func (sv stringValue) Reduce(redactor px.BiMapper) px.Value {
	s := sv.String()
	if len(s) == 0 {
		return undef
	}
	return reduceString(s[1:], sv.At(0), redactor)
}

func (sv stringValue) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduceString(sv.String(), initialValue, redactor)
}

func (sv stringValue) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf(sv.String())
}

func (sv stringValue) ReflectTo(c px.Context, value reflect.Value) {
	switch value.Kind() {
	case reflect.Interface:
		value.Set(sv.Reflect(c))
	case reflect.Ptr:
		s := string(sv)
		value.Set(reflect.ValueOf(&s))
	default:
		value.SetString(string(sv))
	}
}

func (sv stringValue) Reject(predicate px.Predicate) px.List {
	selected := bytes.NewBufferString(``)
	for _, c := range sv.String() {
		if !predicate(stringValue(string(c))) {
			selected.WriteRune(c)
		}
	}
	return stringValue(selected.String())
}

func (sv stringValue) Select(predicate px.Predicate) px.List {
	selected := bytes.NewBufferString(``)
	for _, c := range sv.String() {
		if predicate(stringValue(string(c))) {
			selected.WriteRune(c)
		}
	}
	return stringValue(selected.String())
}

func (sv stringValue) Slice(i int, j int) px.List {
	return stringValue(sv.String()[i:j])
}

func (sv stringValue) Split(pattern *regexp.Regexp) px.List {
	parts := pattern.Split(sv.String(), -1)
	result := make([]px.Value, len(parts))
	for i, s := range parts {
		result[i] = stringValue(s)
	}
	return WrapValues(result)
}

func (sv stringValue) String() string {
	return string(sv)
}

func (sv stringValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), sv.PType())
	val := string(sv)
	switch f.FormatChar() {
	case 's':
		_, err := fmt.Fprintf(b, f.OrigFormat(), val)
		if err != nil {
			panic(err)
		}
	case 'p':
		f.ApplyStringFlags(b, val, true)
	case 'c':
		val = utils.CapitalizeSegment(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 'C':
		val = utils.CapitalizeSegments(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 'u':
		val = strings.ToUpper(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 'd':
		val = strings.ToLower(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 't':
		val = strings.TrimSpace(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	default:
		//noinspection SpellCheckingInspection
		panic(s.UnsupportedFormat(sv.PType(), `cCudspt`, f))
	}
}

func (sv stringValue) ToKey() px.HashKey {
	return px.HashKey(sv.String())
}

func (sv stringValue) ToLower() px.StringValue {
	return stringValue(strings.ToLower(string(sv)))
}

func (sv stringValue) ToUpper() px.StringValue {
	return stringValue(strings.ToUpper(string(sv)))
}

func (sv stringValue) PType() px.Type {
	return &vcStringType{value: string(sv)}
}

func (sv stringValue) Unique() px.List {
	s := sv.String()
	top := len(s)
	if top < 2 {
		return sv
	}

	result := bytes.NewBufferString(``)
	exists := make(map[rune]bool, top)
	for _, c := range s {
		if !exists[c] {
			exists[c] = true
			result.WriteRune(c)
		}
	}
	if result.Len() == len(s) {
		return sv
	}
	return stringValue(result.String())
}

func reduceString(slice string, initialValue px.Value, redactor px.BiMapper) px.Value {
	memo := initialValue
	for _, v := range slice {
		memo = redactor(memo, stringValue(string(v)))
	}
	return memo
}
