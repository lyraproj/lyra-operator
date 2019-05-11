package types

import (
	"io"
	"math"

	"github.com/lyraproj/pcore/px"
)

type TupleType struct {
	size              *IntegerType
	givenOrActualSize *IntegerType
	types             []px.Type
}

var TupleMetaType px.ObjectType

func init() {
	TupleMetaType = newObjectType(`Pcore::TupleType`,
		`Pcore::AnyType {
	attributes => {
		types => Array[Type],
		size_type => {
      type => Optional[Type[Integer]],
      value => undef
    }
  }
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newTupleType2(args...)
		})

	// Go constructor for Tuple instances is registered by ArrayType
}

func DefaultTupleType() *TupleType {
	return tupleTypeDefault
}

func EmptyTupleType() *TupleType {
	return tupleTypeEmpty
}

func NewTupleType(types []px.Type, size *IntegerType) *TupleType {
	var givenOrActualSize *IntegerType
	sz := int64(len(types))
	if size == nil {
		givenOrActualSize = NewIntegerType(sz, sz)
	} else {
		if sz == 0 {
			if *size == *IntegerTypePositive {
				return DefaultTupleType()
			}
			if *size == *IntegerTypeZero {
				return EmptyTupleType()
			}
		}
		givenOrActualSize = size
	}
	return &TupleType{size, givenOrActualSize, types}
}

func newTupleType2(args ...px.Value) *TupleType {
	return tupleFromArgs(false, WrapValues(args))
}

func tupleFromArgs(callable bool, args px.List) *TupleType {
	argc := args.Len()
	if argc == 0 {
		return tupleTypeDefault
	}

	if argc == 1 || argc == 2 {
		if ar, ok := args.At(0).(*Array); ok {
			tupleArgs := ar.AppendTo(make([]px.Value, 0, ar.Len()+argc-1))
			if argc == 2 {
				tupleArgs = append(tupleArgs, args.At(1).(*IntegerType).Parameters()...)
			}
			args = WrapValues(tupleArgs)
			argc = len(tupleArgs)
		}
	}

	var rng, givenOrActualRng *IntegerType
	var ok bool
	var min int64

	last := args.At(argc - 1)
	max := int64(-1)
	if _, ok = last.(*DefaultValue); ok {
		max = math.MaxInt64
	} else if n, ok := toInt(last); ok {
		max = n
	}
	if max >= 0 {
		if argc == 1 {
			rng = NewIntegerType(min, math.MaxInt64)
			argc = 0
		} else {
			if min, ok = toInt(args.At(argc - 2)); ok {
				rng = NewIntegerType(min, max)
				argc -= 2
			} else {
				argc--
				rng = NewIntegerType(max, int64(argc))
			}
		}
		givenOrActualRng = rng
	} else {
		rng = nil
		givenOrActualRng = NewIntegerType(int64(argc), int64(argc))
	}

	if argc == 0 {
		if rng != nil && *rng == *IntegerTypeZero {
			return tupleTypeEmpty
		}
		if callable {
			return &TupleType{rng, rng, []px.Type{DefaultUnitType()}}
		}
		if rng != nil && *rng == *IntegerTypePositive {
			return tupleTypeDefault
		}
		return &TupleType{rng, rng, []px.Type{}}
	}

	var tupleTypes []px.Type
	ok = false
	var failIdx int
	if argc == 1 {
		// One arg can be either array of types or a type
		tupleTypes, failIdx = toTypes(args.Slice(0, 1))
		ok = failIdx < 0
	}

	if !ok {
		tupleTypes, failIdx = toTypes(args.Slice(0, argc))
		if failIdx >= 0 {
			name := `Tuple[]`
			if callable {
				name = `Callable[]`
			}
			panic(illegalArgumentType(name, failIdx, `Type`, args.At(failIdx)))
		}
	}
	return &TupleType{rng, givenOrActualRng, tupleTypes}
}

func (t *TupleType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.size.Accept(v, g)
	for _, c := range t.types {
		c.Accept(v, g)
	}
}

func (t *TupleType) At(i int) px.Value {
	if i >= 0 {
		if i < len(t.types) {
			return t.types[i]
		}
		if int64(i) < t.givenOrActualSize.max {
			return t.types[len(t.types)-1]
		}
	}
	return undef
}

func (t *TupleType) CommonElementType() px.Type {
	top := len(t.types)
	if top == 0 {
		return anyTypeDefault
	}
	cet := t.types[0]
	for idx := 1; idx < top; idx++ {
		cet = commonType(cet, t.types[idx])
	}
	return cet
}

func (t *TupleType) Default() px.Type {
	return tupleTypeDefault
}

func (t *TupleType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*TupleType); ok && len(t.types) == len(ot.types) && px.Equals(t.size, ot.size, g) {
		for idx, col := range t.types {
			if !col.Equals(ot.types[idx], g) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TupleType) Generic() px.Type {
	return NewTupleType(alterTypes(t.types, generalize), t.size)
}

func (t *TupleType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `types`:
		tps := make([]px.Value, len(t.types))
		for i, t := range t.types {
			tps[i] = t
		}
		return WrapValues(tps), true
	case `size_type`:
		if t.size == nil {
			return undef, true
		}
		return t.size, true
	}
	return nil, false
}

func (t *TupleType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o := o.(type) {
	case *ArrayType:
		if !GuardedIsInstance(t.givenOrActualSize, integerValue(o.size.Min()), g) {
			return false
		}
		top := len(t.types)
		if top == 0 {
			return true
		}
		elemType := o.typ
		for idx := 0; idx < top; idx++ {
			if !GuardedIsAssignable(t.types[idx], elemType, g) {
				return false
			}
		}
		return true

	case *TupleType:
		if !(t.size == nil || GuardedIsInstance(t.size, integerValue(o.givenOrActualSize.Min()), g)) {
			return false
		}

		if len(t.types) > 0 {
			top := len(o.types)
			if top == 0 {
				return t.givenOrActualSize.min == 0
			}

			last := len(t.types) - 1
			for idx := 0; idx < top; idx++ {
				myIdx := idx
				if myIdx > last {
					myIdx = last
				}
				if !GuardedIsAssignable(t.types[myIdx], o.types[idx], g) {
					return false
				}
			}
		}
		return true

	default:
		return false
	}
}

func (t *TupleType) IsInstance(v px.Value, g px.Guard) bool {
	if iv, ok := v.(*Array); ok {
		return t.IsInstance2(iv, g)
	}
	return false
}

func (t *TupleType) IsInstance2(vs px.List, g px.Guard) bool {
	osz := vs.Len()
	if !t.givenOrActualSize.IsInstance3(osz) {
		return false
	}

	last := len(t.types) - 1
	if last < 0 {
		return true
	}

	tdx := 0
	for idx := 0; idx < osz; idx++ {
		if !GuardedIsInstance(t.types[tdx], vs.At(idx), g) {
			return false
		}
		if tdx < last {
			tdx++
		}
	}
	return true
}

func (t *TupleType) IsInstance3(vs []px.Value, g px.Guard) bool {
	osz := len(vs)
	if !t.givenOrActualSize.IsInstance3(osz) {
		return false
	}

	last := len(t.types) - 1
	if last < 0 {
		return true
	}

	tdx := 0
	for idx := 0; idx < osz; idx++ {
		if !GuardedIsInstance(t.types[tdx], vs[idx], g) {
			return false
		}
		if tdx < last {
			tdx++
		}
	}
	return true
}

func (t *TupleType) MetaType() px.ObjectType {
	return TupleMetaType
}

func (t *TupleType) Name() string {
	return `Tuple`
}

func (t *TupleType) Resolve(c px.Context) px.Type {
	rts := make([]px.Type, len(t.types))
	for i, ts := range t.types {
		rts[i] = resolve(c, ts)
	}
	t.types = rts
	return t
}

func (t *TupleType) CanSerializeAsString() bool {
	for _, v := range t.types {
		if !canSerializeAsString(v) {
			return false
		}
	}
	return true
}

func (t *TupleType) SerializationString() string {
	return t.String()
}

func (t *TupleType) Size() *IntegerType {
	return t.givenOrActualSize
}

func (t *TupleType) String() string {
	return px.ToString2(t, None)
}

func (t *TupleType) Parameters() []px.Value {
	top := len(t.types)
	params := make([]px.Value, 0, top+2)
	for _, c := range t.types {
		params = append(params, c)
	}
	if !(t.size == nil || top == 0 && *t.size == *IntegerTypePositive) {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *TupleType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TupleType) PType() px.Type {
	return &TypeType{t}
}

func (t *TupleType) Types() []px.Type {
	return t.types
}

var tupleTypeDefault = &TupleType{IntegerTypePositive, IntegerTypePositive, []px.Type{}}
var tupleTypeEmpty = &TupleType{IntegerTypeZero, IntegerTypeZero, []px.Type{}}
