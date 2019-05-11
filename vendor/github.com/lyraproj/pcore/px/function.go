package px

type (
	InvokableValue interface {
		Value

		Call(c Context, block Lambda, args ...Value) Value
	}

	Parameter interface {
		Value

		Name() string

		Type() Type

		HasValue() bool

		Value() Value

		CapturesRest() bool
	}

	Lambda interface {
		InvokableValue

		Parameters() []Parameter

		Signature() Signature
	}

	Function interface {
		InvokableValue

		Dispatchers() []Lambda

		Name() string
	}

	ResolvableFunction interface {
		Name() string
		Resolve(c Context) Function
	}

	DispatchFunction func(c Context, args []Value) Value

	DispatchFunctionWithBlock func(c Context, args []Value, block Lambda) Value

	LocalTypes interface {
		Type(name string, decl string)
		Type2(name string, tp Type)
	}

	// Dispatch is a builder to build function dispatchers (Lambdas)
	Dispatch interface {
		// Name returns the name of the owner function
		Name() string

		Param(typeString string)
		Param2(puppetType Type)

		OptionalParam(typeString string)
		OptionalParam2(puppetType Type)

		RepeatedParam(typeString string)
		RepeatedParam2(puppetType Type)

		RequiredRepeatedParam(typeString string)
		RequiredRepeatedParam2(puppetType Type)

		Block(typeString string)
		Block2(puppetType Type)

		OptionalBlock(typeString string)
		OptionalBlock2(puppetType Type)

		Returns(typeString string)
		Returns2(puppetType Type)

		Function(f DispatchFunction)
		Function2(f DispatchFunctionWithBlock)
	}

	Signature interface {
		Type

		CallableWith(args []Value, block Lambda) bool

		ParametersType() Type

		ReturnType() Type

		// BlockType returns a Callable, Optional[Callable], or nil to denote if a
		// block is required, optional, or invalid
		BlockType() Type

		// BlockName will typically return the string "block"
		BlockName() string

		// ParameterNames returns the names of the parameters. Will return the strings "1", "2", etc.
		// for unnamed parameters.
		ParameterNames() []string
	}

	DispatchCreator func(db Dispatch)

	LocalTypesCreator func(lt LocalTypes)
)

var BuildFunction func(name string, localTypes LocalTypesCreator, creators []DispatchCreator) ResolvableFunction

var NewGoFunction func(name string, creators ...DispatchCreator)

var NewGoFunction2 func(name string, localTypes LocalTypesCreator, creators ...DispatchCreator)

var NewGoConstructor func(typeName string, creators ...DispatchCreator)

var MakeGoAllocator func(allocFunc DispatchFunction) Lambda

var NewGoConstructor2 func(typeName string, localTypes LocalTypesCreator, creators ...DispatchCreator)

var NewParameter func(name string, typ Type, value Value, capturesRest bool) Parameter

var MakeGoConstructor func(typeName string, creators ...DispatchCreator) ResolvableFunction

var MakeGoConstructor2 func(typeName string, localTypes LocalTypesCreator, creators ...DispatchCreator) ResolvableFunction
