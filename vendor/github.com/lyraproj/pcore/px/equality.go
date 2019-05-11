package px

import "reflect"

type (
	visit struct {
		a1 interface{}
		a2 interface{}
	}

	// Guard helps tracking endless recursion. The comparison algorithm assumes that all checks in progress
	// are true when it encounters them again. Visited comparisons are stored in a map
	// indexed by visit.
	//
	// (algorithm copied from golang reflect/deepequal.go)
	Guard map[visit]bool

	// Equality is implemented by values that can be compared for equality with other values
	Equality interface {
		// Returns true if the receiver is equal to the given value, false otherwise.
		Equals(value interface{}, guard Guard) bool
	}
)

// Seen returns true if the combination of the given values has been seen by this guard and if not, registers
// the combination so that true is returned the next time the same values are given.
func (g Guard) Seen(a, b interface{}) bool {
	v := visit{a, b}
	if _, ok := g[v]; ok {
		return true
	}
	g[v] = true
	return false
}

// Equals will compare two values for equality. If the first value implements the Equality interface, then
// the that interface is used. If the first value is a primitive, then the primitive will be compared using
// ==. The default behavior is to delegate to reflect.DeepEqual.
//
// The comparison is optionally guarded from endless recursion by passing a Guard instance
func Equals(a interface{}, b interface{}, g Guard) bool {
	switch a := a.(type) {
	case nil:
		return b == nil
	case Equality:
		return a.(Equality).Equals(b, g)
	case bool:
		bs, ok := b.(bool)
		return ok && a == bs
	case int:
		bs, ok := b.(int)
		return ok && a == bs
	case int64:
		bs, ok := b.(int64)
		return ok && a == bs
	case float64:
		bs, ok := b.(float64)
		return ok && a == bs
	case string:
		bs, ok := b.(string)
		return ok && a == bs
	default:
		return reflect.DeepEqual(a, b)
	}
}

// IncludesAll returns true if the given slice a contains all values
// in the given slice b.
func IncludesAll(a interface{}, b interface{}, g Guard) bool {
	ra := reflect.ValueOf(a)
	la := ra.Len()
	rb := reflect.ValueOf(b)
	lb := rb.Len()
	for ia := 0; ia < la; ia++ {
		v := ra.Index(ia).Interface()
		found := false
		for ib := 0; ib < lb; ib++ {
			ov := rb.Index(ib).Interface()
			if Equals(ov, v, g) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// IndexFrom returns the index in the given slice of the first occurrence of an
// element for which Equals with the given value starting the comparison at the
// given startPos. The value of -1 is returned when no such element is found.
func IndexFrom(slice interface{}, value interface{}, startPos int, g Guard) int {
	ra := reflect.ValueOf(slice)
	la := ra.Len()
	for idx := startPos; idx < la; idx++ {
		if Equals(ra.Index(idx).Interface(), value, g) {
			return idx
		}
	}
	return -1
}

// ReverseIndexFrom returns the index in the given slice of the last occurrence of an
// element for which Equals with the given value between position zero and the given
// endPos. If endPos is -1 it will be set to the length of the slice.
//
// The value of -1 is returned when no such element is found.
func ReverseIndexFrom(slice interface{}, value interface{}, endPos int, g Guard) int {
	ra := reflect.ValueOf(slice)
	top := ra.Len()
	idx := top - 1
	if endPos >= 0 && endPos < idx {
		idx = endPos
	}
	for ; idx >= 0; idx-- {
		if Equals(ra.Index(idx).Interface(), value, g) {
			return idx
		}
	}
	return -1
}

// PuppetEquals is like Equals but:
//   int and float values with same value are considered equal
//   string comparisons are case insensitive
var PuppetEquals func(a, b Value) bool

// PuppetMatch tests if the LHS matches the RHS pattern expression and returns a Boolean result
//
// When the RHS is a Type:
//
//   the match is true if the LHS is an instance of the type
//   No match variables are set in this case.
//
// When the RHS is a SemVerRange:
//
//   the match is true if the LHS is a SemVer, and the version is within the range
//   the match is true if the LHS is a String representing a SemVer, and the version is within the range
//   an error is raised if the LHS is neither a String with a valid SemVer representation, nor a SemVer.
//   otherwise the result is false (not in range).
//
// When the RHS is not a Type:
//
//   If the RHS evaluates to a String a new Regular Expression is created with the string value as its pattern.
//   If the RHS is not a Regexp (after string conversion) an error is raised.
//   If the LHS is not a String an error is raised. (Note, Numeric values are not converted to String automatically because of unknown radix).
var PuppetMatch func(lhs, rhs Value) bool
