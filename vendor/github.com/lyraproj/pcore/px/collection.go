package px

// ToArray returns a List consisting of the given elements
var ToArray func(elements []Value) List

// All passes each of the given elements to the given predicate. It returns true if the predicate
// never returns false.
func All(elements []Value, predicate Predicate) bool {
	for _, elem := range elements {
		if !predicate(elem) {
			return false
		}
	}
	return true
}

// Any passes each of the given elements to the given predicate until the predicate returns true. The
// function then returns true. The function returns false when no predicate returns true.
func Any(elements []Value, predicate Predicate) bool {
	for _, elem := range elements {
		if predicate(elem) {
			return true
		}
	}
	return false
}

// Each passes each of the given elements to the given consumer
func Each(elements []Value, consumer Consumer) {
	for _, elem := range elements {
		consumer(elem)
	}
}

// Find passes each of the given elements to the given predicate and returns the first element for
// which the predicate returns true together with a boolean true. The function returns nil, false
// when no predicate returns true.
func Find(elements []Value, predicate Predicate) (Value, bool) {
	for _, elem := range elements {
		if predicate(elem) {
			return elem, true
		}
	}
	return nil, false
}

// Map passes each of the given elements to the given mapper and builds a new slice from the
// mapper return values. The new slice is returned.
func Map(elements []Value, mapper Mapper) []Value {
	result := make([]Value, len(elements))
	for idx, elem := range elements {
		result[idx] = mapper(elem)
	}
	return result
}

// Reduce combines all elements of the given slice by applying a binary operation. For each
// element in teh slice, the reductor is passed an accumulator value (memo) and the element.
// The result becomes the new value for the memo. At the end of iteration, the final memo
// is returned.
func Reduce(elements []Value, memo Value, reductor BiMapper) Value {
	for _, elem := range elements {
		memo = reductor(memo, elem)
	}
	return memo
}

// Select passes each of the given elements to the given predicate and returns a new slice
// containing those elements for which  predicate returned true.
func Select(elements []Value, predicate Predicate) []Value {
	result := make([]Value, 0, 8)
	for _, elem := range elements {
		if predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}

// Select passes each of the given elements to the given predicate and returns a new slice
// containing those elements for which  predicate returned false.
func Reject(elements []Value, predicate Predicate) []Value {
	result := make([]Value, 0, 8)
	for _, elem := range elements {
		if !predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}
