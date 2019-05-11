package pximpl

import (
	"fmt"
	"io"
	"math"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/pcore/utils"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type (
	typeDecl struct {
		name string
		decl string
		tp   px.Type
	}

	functionBuilder struct {
		name             string
		localTypeBuilder *localTypeBuilder
		dispatchers      []*dispatchBuilder
	}

	localTypeBuilder struct {
		localTypes []*typeDecl
	}

	dispatchBuilder struct {
		fb            *functionBuilder
		min           int64
		max           int64
		types         []px.Type
		blockType     px.Type
		optionalBlock bool
		returnType    px.Type
		function      px.DispatchFunction
		function2     px.DispatchFunctionWithBlock
	}

	goFunction struct {
		name        string
		dispatchers []px.Lambda
	}

	lambda struct {
		signature *types.CallableType
	}

	goLambda struct {
		lambda
		function px.DispatchFunction
	}

	goLambdaWithBlock struct {
		lambda
		function px.DispatchFunctionWithBlock
	}
)

func parametersFromSignature(s px.Signature) []px.Parameter {
	paramNames := s.ParameterNames()
	count := len(paramNames)
	tuple := s.ParametersType().(*types.TupleType)
	tz := tuple.Size()
	capture := -1
	if tz.Max() > int64(count) {
		capture = count - 1
	}
	paramTypes := s.ParametersType().(*types.TupleType).Types()
	ps := make([]px.Parameter, len(paramNames))
	for i, paramName := range paramNames {
		ps[i] = newParameter(paramName, paramTypes[i], nil, i == capture)
	}
	return ps
}

func (l *lambda) Equals(other interface{}, guard px.Guard) bool {
	if ol, ok := other.(*lambda); ok {
		return l.signature.Equals(ol.signature, guard)
	}
	return false
}

func (l *lambda) String() string {
	return `lambda`
}

func (l *lambda) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	utils.WriteString(bld, `lambda`)
}

func (l *lambda) PType() px.Type {
	return l.signature
}

func (l *lambda) Signature() px.Signature {
	return l.signature
}

func (l *goLambda) Call(c px.Context, block px.Lambda, args ...px.Value) px.Value {
	return l.function(c, args)
}

func (l *goLambda) Parameters() []px.Parameter {
	return parametersFromSignature(l.signature)
}

func (l *goLambdaWithBlock) Call(c px.Context, block px.Lambda, args ...px.Value) px.Value {
	return l.function(c, args, block)
}

func (l *goLambdaWithBlock) Parameters() []px.Parameter {
	return parametersFromSignature(l.signature)
}

var emptyTypeBuilder = &localTypeBuilder{[]*typeDecl{}}

func buildFunction(name string, localTypes px.LocalTypesCreator, creators []px.DispatchCreator) px.ResolvableFunction {
	lt := emptyTypeBuilder
	if localTypes != nil {
		lt = &localTypeBuilder{make([]*typeDecl, 0, 8)}
		localTypes(lt)
	}

	fb := &functionBuilder{name: name, localTypeBuilder: lt, dispatchers: make([]*dispatchBuilder, len(creators))}
	dbs := fb.dispatchers
	fb.dispatchers = dbs
	for idx, creator := range creators {
		dbs[idx] = fb.newDispatchBuilder()
		creator(dbs[idx])
	}
	return fb
}

func (fb *functionBuilder) newDispatchBuilder() *dispatchBuilder {
	return &dispatchBuilder{fb: fb, types: make([]px.Type, 0, 8), min: 0, max: 0, optionalBlock: false, blockType: nil, returnType: nil}
}

func (fb *functionBuilder) Name() string {
	return fb.name
}

func (fb *functionBuilder) Resolve(c px.Context) px.Function {
	ds := make([]px.Lambda, len(fb.dispatchers))

	if tl := len(fb.localTypeBuilder.localTypes); tl > 0 {
		localLoader := px.NewParentedLoader(c.Loader())
		c.DoWithLoader(localLoader, func() {
			te := make([]px.Type, 0, tl)
			for _, td := range fb.localTypeBuilder.localTypes {
				if td.tp == nil {
					v, err := types.Parse(td.decl)
					if err != nil {
						panic(err)
					}
					if dt, ok := v.(*types.DeferredType); ok {
						te = append(te, types.NamedType(px.RuntimeNameAuthority, td.name, dt))
					}
				} else {
					localLoader.SetEntry(px.NewTypedName(px.NsType, td.name), px.NewLoaderEntry(td.tp, nil))
				}
			}

			if len(te) > 0 {
				px.AddTypes(c, te...)
			}
			for i, d := range fb.dispatchers {
				ds[i] = d.createDispatch(c)
			}
		})
	} else {
		for i, d := range fb.dispatchers {
			ds[i] = d.createDispatch(c)
		}
	}
	return &goFunction{fb.name, ds}
}

func (tb *localTypeBuilder) Type(name string, decl string) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, decl, nil})
}

func (tb *localTypeBuilder) Type2(name string, tp px.Type) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, ``, tp})
}

func (db *dispatchBuilder) createDispatch(c px.Context) px.Lambda {
	for idx, tp := range db.types {
		if trt, ok := tp.(*types.TypeReferenceType); ok {
			db.types[idx] = c.ParseType(trt.TypeString())
		}
	}
	if r, ok := db.blockType.(*types.TypeReferenceType); ok {
		db.blockType = c.ParseType(r.TypeString())
	}
	if db.optionalBlock {
		db.blockType = types.NewOptionalType(db.blockType)
	}
	if r, ok := db.returnType.(*types.TypeReferenceType); ok {
		db.returnType = c.ParseType(r.TypeString())
	}
	if db.function2 == nil {
		return &goLambda{lambda{types.NewCallableType(types.NewTupleType(db.types, types.NewIntegerType(db.min, db.max)), db.returnType, nil)}, db.function}
	}
	return &goLambdaWithBlock{lambda{types.NewCallableType(types.NewTupleType(db.types, types.NewIntegerType(db.min, db.max)), db.returnType, db.blockType)}, db.function2}
}

func (db *dispatchBuilder) Name() string {
	return db.fb.name
}

func (db *dispatchBuilder) Param(tp string) {
	db.Param2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Param2(tp px.Type) {
	db.assertNotAfterRepeated()
	if db.min < db.max {
		panic(`Required parameters must not come after optional parameters in a dispatch`)
	}
	db.types = append(db.types, tp)
	db.min++
	db.max++
}

func (db *dispatchBuilder) OptionalParam(tp string) {
	db.OptionalParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) OptionalParam2(tp px.Type) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max++
}

func (db *dispatchBuilder) RepeatedParam(tp string) {
	db.RepeatedParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RepeatedParam2(tp px.Type) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) RequiredRepeatedParam(tp string) {
	db.RequiredRepeatedParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RequiredRepeatedParam2(tp px.Type) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.min++
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) Block(tp string) {
	db.Block2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Block2(tp px.Type) {
	if db.returnType != nil {
		panic(`Block specified more than once`)
	}
	db.blockType = tp
}

func (db *dispatchBuilder) OptionalBlock(tp string) {
	db.OptionalBlock2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) OptionalBlock2(tp px.Type) {
	db.Block2(tp)
	db.optionalBlock = true
}

func (db *dispatchBuilder) Returns(tp string) {
	db.Returns2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Returns2(tp px.Type) {
	if db.returnType != nil {
		panic(`Returns specified more than once`)
	}
	db.returnType = tp
}

func (db *dispatchBuilder) Function(df px.DispatchFunction) {
	if _, ok := db.blockType.(*types.CallableType); ok {
		panic(`Dispatch requires a block. Use FunctionWithBlock`)
	}
	db.function = df
}

func (db *dispatchBuilder) Function2(df px.DispatchFunctionWithBlock) {
	if db.blockType == nil {
		panic(`Dispatch does not expect a block. Use Function instead of FunctionWithBlock`)
	}
	db.function2 = df
}

func (db *dispatchBuilder) assertNotAfterRepeated() {
	if db.max == math.MaxInt64 {
		panic(`Repeated parameters can only occur last in a dispatch`)
	}
}

func (f *goFunction) Call(c px.Context, block px.Lambda, args ...px.Value) px.Value {
	for _, d := range f.dispatchers {
		if d.Signature().CallableWith(args, block) {
			return d.Call(c, block, args...)
		}
	}
	panic(px.Error(px.IllegalArguments, issue.H{`function`: f.name, `message`: px.DescribeSignatures(signatures(f.dispatchers), types.WrapValues(args).DetailedType(), block)}))
}

func signatures(lambdas []px.Lambda) []px.Signature {
	s := make([]px.Signature, len(lambdas))
	for i, l := range lambdas {
		s[i] = l.Signature()
	}
	return s
}

func (f *goFunction) Dispatchers() []px.Lambda {
	return f.dispatchers
}

func (f *goFunction) Name() string {
	return f.name
}

func (f *goFunction) Equals(other interface{}, g px.Guard) bool {
	dc := len(f.dispatchers)
	if of, ok := other.(*goFunction); ok && f.name == of.name && dc == len(of.dispatchers) {
		for i := 0; i < dc; i++ {
			if !f.dispatchers[i].Equals(of.dispatchers[i], g) {
				return false
			}
		}
		return true
	}
	return false
}

func (f *goFunction) String() string {
	return fmt.Sprintf(`function %s`, f.name)
}

func (f *goFunction) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	utils.WriteString(bld, `function `)
	utils.WriteString(bld, f.name)
}

func (f *goFunction) PType() px.Type {
	top := len(f.dispatchers)
	variants := make([]px.Type, top)
	for idx := 0; idx < top; idx++ {
		variants[idx] = f.dispatchers[idx].PType()
	}
	return types.NewVariantType(variants...)
}

func init() {
	px.BuildFunction = buildFunction

	px.NewGoFunction = func(name string, creators ...px.DispatchCreator) {
		px.RegisterGoFunction(buildFunction(name, nil, creators))
	}

	px.NewGoFunction2 = func(name string, localTypes px.LocalTypesCreator, creators ...px.DispatchCreator) {
		px.RegisterGoFunction(buildFunction(name, localTypes, creators))
	}

	px.MakeGoAllocator = func(allocFunc px.DispatchFunction) px.Lambda {
		return &goLambda{lambda{types.NewCallableType(types.EmptyTupleType(), nil, nil)}, allocFunc}
	}

	px.MakeGoConstructor = func(typeName string, creators ...px.DispatchCreator) px.ResolvableFunction {
		return buildFunction(typeName, nil, creators)
	}

	px.MakeGoConstructor2 = func(typeName string, localTypes px.LocalTypesCreator, creators ...px.DispatchCreator) px.ResolvableFunction {
		return buildFunction(typeName, localTypes, creators)
	}

	px.NewGoFunction(`new`,
		func(d px.Dispatch) {
			d.Param(`Variant[Type,String]`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return px.NewWithBlock(c, args[0], args[1:], block)
			})
		},
	)
}
