package types

import (
	"io"
	"strconv"

	"github.com/lyraproj/pcore/px"
)

type CallableType struct {
	paramsType px.Type
	returnType px.Type
	blockType  px.Type // Callable or Optional[Callable]
}

var CallableMetaType px.ObjectType

func init() {
	CallableMetaType = newObjectType(`Pcore::CallableType`,
		`Pcore::AnyType {
  attributes => {
    param_types => {
      type => Optional[Type[Tuple]],
      value => undef
    },
    block_type => {
      type => Optional[Type[Callable]],
      value => undef
    },
    return_type => {
      type => Optional[Type],
      value => undef
    }
  }
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newCallableType2(args...)
		})
}

func DefaultCallableType() *CallableType {
	return callableTypeDefault
}

func NewCallableType(paramsType px.Type, returnType px.Type, blockType px.Type) *CallableType {
	return &CallableType{paramsType, returnType, blockType}
}

func newCallableType2(args ...px.Value) *CallableType {
	return newCallableType3(WrapValues(args))
}

func newCallableType3(args px.List) *CallableType {
	argc := args.Len()
	if argc == 0 {
		return DefaultCallableType()
	}

	first := args.At(0)
	if tv, ok := first.(*TupleType); ok {
		var returnType px.Type
		var blockType px.Type
		if argc > 1 {
			blockType, ok = args.At(1).(px.Type)
			if argc > 2 {
				returnType, ok = args.At(2).(px.Type)
			}
		}
		if ok {
			return &CallableType{tv, returnType, blockType}
		}
	}

	var (
		rt    px.Type
		block px.Type
		ok    bool
	)

	if argc == 1 || argc == 2 {
		// check for [[params, block], return]
		var iv px.List
		if iv, ok = first.(px.List); ok {
			if argc == 2 {
				if rt, ok = args.At(1).(px.Type); !ok {
					panic(illegalArgumentType(`Callable[]`, 1, `Type`, args.At(1)))
				}
			}
			argc = iv.Len()
			args = iv
		}
	}

	last := args.At(argc - 1)
	block, ok = last.(*CallableType)
	if !ok {
		block = nil
		var ob *OptionalType
		if ob, ok = last.(*OptionalType); ok {
			if _, ok = ob.typ.(*CallableType); ok {
				block = ob
			}
		}
	}
	if ok {
		argc--
		args = args.Slice(0, argc)
	}
	return NewCallableType(tupleFromArgs(true, args), rt, block)
}

func (t *CallableType) BlockType() px.Type {
	if t.blockType == nil {
		return nil // Return untyped nil
	}
	return t.blockType
}

func (t *CallableType) CallableWith(args []px.Value, block px.Lambda) bool {
	if block != nil {
		cb := t.blockType
		switch ca := cb.(type) {
		case nil:
			return false
		case *OptionalType:
			cb = ca.ContainedType()
		}
		if block.PType() == nil {
			return false
		}
		if !isAssignable(block.PType(), cb) {
			return false
		}
	} else if t.blockType != nil && !isAssignable(t.blockType, anyTypeDefault) {
		// Required block but non provided
		return false
	}
	if pt, ok := t.paramsType.(*TupleType); ok {
		return pt.IsInstance3(args, nil)
	}
	return true
}

func (t *CallableType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	if t.paramsType != nil {
		t.paramsType.Accept(v, g)
	}
	if t.blockType != nil {
		t.blockType.Accept(v, g)
	}
	if t.returnType != nil {
		t.returnType.Accept(v, g)
	}
}

func (t *CallableType) BlockName() string {
	return `block`
}

func (t *CallableType) CanSerializeAsString() bool {
	return canSerializeAsString(t.paramsType) && canSerializeAsString(t.blockType) && canSerializeAsString(t.returnType)
}

func (t *CallableType) SerializationString() string {
	return t.String()
}

func (t *CallableType) Default() px.Type {
	return callableTypeDefault
}

func (t *CallableType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*CallableType)
	return ok
}

func (t *CallableType) Generic() px.Type {
	return callableTypeDefault
}

func (t *CallableType) Get(key string) (px.Value, bool) {
	switch key {
	case `param_types`:
		if t.paramsType == nil {
			return px.Undef, true
		}
		return t.paramsType, true
	case `return_type`:
		if t.returnType == nil {
			return px.Undef, true
		}
		return t.returnType, true
	case `block_type`:
		if t.blockType == nil {
			return px.Undef, true
		}
		return t.blockType, true
	default:
		return nil, false
	}
}

func (t *CallableType) IsAssignable(o px.Type, g px.Guard) bool {
	oc, ok := o.(*CallableType)
	if !ok {
		return false
	}
	if t.returnType == nil && t.paramsType == nil && t.blockType == nil {
		return true
	}

	if t.returnType != nil {
		or := oc.returnType
		if or == nil {
			or = anyTypeDefault
		}
		if !isAssignable(t.returnType, or) {
			return false
		}
	}

	// NOTE: these tests are made in reverse as it is calling the callable that is constrained
	// (it's lower bound), not its upper bound
	if oc.paramsType != nil && (t.paramsType == nil || !isAssignable(oc.paramsType, t.paramsType)) {
		return false
	}

	if t.blockType == nil {
		return oc.blockType == nil
	}
	if oc.blockType == nil {
		return false
	}
	return isAssignable(oc.blockType, t.blockType)
}

func (t *CallableType) IsInstance(o px.Value, g px.Guard) bool {
	if l, ok := o.(px.Lambda); ok {
		return isAssignable(t, l.PType())
	}
	// TODO: Maybe check Go func using reflection
	return false
}

func (t *CallableType) MetaType() px.ObjectType {
	return CallableMetaType
}

func (t *CallableType) Name() string {
	return `Callable`
}

func (t *CallableType) ParameterNames() []string {
	if pt, ok := t.paramsType.(*TupleType); ok {
		n := len(pt.types)
		r := make([]string, 0, n)
		for i := 0; i < n; {
			i++
			r = append(r, strconv.Itoa(i))
		}
		return r
	}
	return []string{}
}

func (t *CallableType) Parameters() (params []px.Value) {
	if *t == *callableTypeDefault {
		return px.EmptyValues
	}
	if pt, ok := t.paramsType.(*TupleType); ok {
		tupleParams := pt.Parameters()
		if len(tupleParams) == 0 {
			params = make([]px.Value, 0)
		} else {
			params = px.Select(tupleParams, func(p px.Value) bool { _, ok := p.(*UnitType); return !ok })
		}
	} else {
		params = make([]px.Value, 0)
	}
	if t.blockType != nil {
		params = append(params, t.blockType)
	}
	if t.returnType != nil {
		params = []px.Value{WrapValues(params), t.returnType}
	}
	return params
}

func (t *CallableType) ParametersType() px.Type {
	if t.paramsType == nil {
		return nil // Return untyped nil
	}
	return t.paramsType
}

func (t *CallableType) Resolve(c px.Context) px.Type {
	if t.paramsType != nil {
		t.paramsType = resolve(c, t.paramsType).(*TupleType)
	}
	if t.returnType != nil {
		t.returnType = resolve(c, t.returnType)
	}
	if t.blockType != nil {
		t.blockType = resolve(c, t.blockType)
	}
	return t
}

func (t *CallableType) ReturnType() px.Type {
	return t.returnType
}

func (t *CallableType) String() string {
	return px.ToString2(t, None)
}

func (t *CallableType) PType() px.Type {
	return &TypeType{t}
}

func (t *CallableType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

var callableTypeDefault = &CallableType{paramsType: nil, blockType: nil, returnType: nil}
