package px

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
)

type (
	RDetect map[interface{}]bool

	Value interface {
		fmt.Stringer
		Equality
		ToString(bld io.Writer, format FormatContext, g RDetect)
		PType() Type
	}

	// Comparator returns true when a is less than b.
	Comparator func(a, b Value) bool

	Object interface {
		Value
		Initialize(c Context, arguments []Value)
		InitFromHash(c Context, hash OrderedMap)
	}

	ReadableObject interface {
		Get(key string) (value Value, ok bool)
	}

	// CallableObject is implemented by PuppetObjects that have functions
	CallableObject interface {
		Call(c Context, method ObjFunc, args []Value, block Lambda) (result Value, ok bool)
	}

	PuppetObject interface {
		Value
		ReadableObject

		InitHash() OrderedMap
	}

	DetailedTypeValue interface {
		Value
		DetailedType() Type
	}

	Sized interface {
		Value
		Len() int
		IsEmpty() bool
	}

	InterfaceValue interface {
		Value
		Interface() interface{}
	}

	Arrayable interface {
		AsArray() List
	}

	Indexed interface {
		Sized
		At(index int) Value
		ElementType() Type
		IsHashStyle() bool
	}

	IteratorValue interface {
		Value
		Arrayable
		ElementType() Type
		Next() (Value, bool)
	}

	// List represents an Array. The iterative methods will not catch break exceptions. If
	//	// that is desired, then use an Iterator instead.
	List interface {
		Indexed
		Add(Value) List
		AddAll(List) List
		All(predicate Predicate) bool
		Any(predicate Predicate) bool
		AppendTo(slice []Value) []Value
		Delete(Value) List
		DeleteAll(List) List
		Each(Consumer)
		EachSlice(int, SliceConsumer)
		EachWithIndex(consumer IndexedConsumer)
		Find(predicate Predicate) (Value, bool)
		Flatten() List
		Map(mapper Mapper) List
		Select(predicate Predicate) List
		Slice(i int, j int) List
		Reduce(redactor BiMapper) Value
		Reduce2(initialValue Value, redactor BiMapper) Value
		Reject(predicate Predicate) List
		Unique() List
	}

	SortableList interface {
		List
		Sort(comparator Comparator) List
	}

	HashKey string

	HashKeyValue interface {
		ToKey() HashKey
	}

	StreamHashKeyValue interface {
		ToKey(b *bytes.Buffer)
	}

	MapEntry interface {
		Value
		Key() Value
		Value() Value
	}

	Keyed interface {
		Get(key Value) (Value, bool)
	}

	// OrderedMap represents a Hash. The iterative methods will not catch break exceptions. If
	// that is desired, then use an Iterator instead.
	OrderedMap interface {
		List
		Keyed
		AllPairs(BiPredicate) bool
		AnyPair(BiPredicate) bool
		AllKeysAreStrings() bool
		Entries() List
		EachKey(Consumer)
		EachPair(BiConsumer)
		EachValue(Consumer)

		Get2(key Value, dflt Value) Value
		Get3(key Value, dflt Producer) Value
		Get4(key string) (Value, bool)
		Get5(key string, dflt Value) Value
		Get6(key string, dflt Producer) Value

		// GetEntry returns the entry that represents the mapping between
		// the given key and its value
		GetEntry(key string) (entry MapEntry, found bool)

		// GetEntryFold retuns the first entry that has a key which, in string form
		// equals the given key using case insensitive comparison
		GetEntryFold(key string) (entry MapEntry, found bool)

		IncludesKey(o Value) bool

		IncludesKey2(o string) bool

		Keys() List

		// MapEntries returns a new OrderedMap with both keys and values
		// converted using the given mapper function
		MapEntries(mapper EntryMapper) OrderedMap

		// MapValues returns a new OrderedMap with the exact same keys as
		// before but where each value has been converted using the given
		// mapper function
		MapValues(mapper Mapper) OrderedMap

		Merge(OrderedMap) OrderedMap

		Values() List
		SelectPairs(BiPredicate) OrderedMap
		RejectPairs(BiPredicate) OrderedMap
	}

	Number interface {
		Value
		Int() int64
		Float() float64
	}

	Boolean interface {
		Value
		Bool() bool
	}

	Integer interface {
		Number
		Abs() int64
	}

	Float interface {
		Number
		Abs() float64
	}

	StringValue interface {
		List
		Split(pattern *regexp.Regexp) List
		ToLower() StringValue
		ToUpper() StringValue
		EqualsIgnoreCase(Value) bool
	}
)

var EmptyArray List
var EmptyMap OrderedMap
var EmptyString Value
var EmptyValues []Value
var Undef Value

var SingletonMap func(string, Value) OrderedMap
var DetailedValueType func(value Value) Type
var GenericValueType func(value Value) Type
var ToKey func(value Value) HashKey
var IsTruthy func(tv Value) bool

var ToInt func(v Value) (int64, bool)
var ToFloat func(v Value) (float64, bool)

// StringElements returns a slice containing each element in the given list as a string
func StringElements(l List) []string {
	ss := make([]string, l.Len())
	l.EachWithIndex(func(e Value, i int) {
		ss[i] = e.String()
	})
	return ss
}

func ToString(t Value) string {
	return ToString2(t, DefaultFormatContext)
}

func ToPrettyString(t Value) string {
	return ToString2(t, Pretty)
}

func ToString2(t Value, format FormatContext) string {
	bld := bytes.NewBufferString(``)
	t.ToString(bld, format, nil)
	return bld.String()
}

func ToString3(t Value, writer io.Writer) {
	ToString4(t, DefaultFormatContext, writer)
}

func ToString4(t Value, format FormatContext, writer io.Writer) {
	t.ToString(writer, format, nil)
}

func CopyValues(src []Value) []Value {
	dst := make([]Value, len(src))
	copy(dst, src)
	return dst
}
