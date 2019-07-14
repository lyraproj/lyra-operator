package wfe

import (
	"fmt"
	"sync/atomic"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/wf"
	"github.com/lyraproj/wfe/api"
	"github.com/lyraproj/wfe/service"
)

const maxParallel = 100

type iterator struct {
	api.Step
	over       px.Value
	variables  []px.Parameter
	resultName string
}

func Iterator(c px.Context, def serviceapi.Definition) api.Step {
	over := def.Properties().Get5(`over`, px.Undef)
	variables := getParameters(`variable`, def.Properties())
	if len(variables) == 0 {
		variables = getParameters(`variables`, def.Properties())
	}
	style := wf.NewIterationStyle(service.GetStringProperty(def, `iterationStyle`))
	step := CreateStep(c, service.GetProperty(def, `producer`, serviceapi.DefinitionMetaType).(serviceapi.Definition))
	var resultName string
	if into, ok := def.Properties().Get4(`into`); ok {
		resultName = into.String()
	} else {
		resultName = wf.LeafName(def.Identifier().Name())
	}
	switch style {
	case wf.IterationStyleEach:
		return NewEach(step, resultName, over, variables)
	case wf.IterationStyleEachPair:
		return NewEachPair(step, resultName, over, variables)
	case wf.IterationStyleRange:
		return NewRange(step, resultName, over, variables)
	case wf.IterationStyleTimes:
		return NewTimes(step, resultName, over, variables)
	default:
		panic(px.Error(wf.IllegalIterationStyle, issue.H{`style`: style.String()}))
	}
}

func (it *iterator) IterationStyle() wf.IterationStyle {
	panic("implement me")
}

func (it *iterator) Over() px.Value {
	return it.over
}

func (it *iterator) Producer() api.Step {
	return it.Step
}

// Parameters returns the Parameters declared for the stateHandler + Over() and - Variables
func (it *iterator) Parameters() []px.Parameter {
	parameters := it.Producer().Parameters()
	all := make([]px.Parameter, 0, len(parameters)-len(it.variables))
nextParameters:
	for _, in := range parameters {
		for _, v := range it.variables {
			if in.Name() == v.Name() {
				continue nextParameters
			}
		}
		all = append(all, in)
	}
	return all
}

// Returns returns one parameter named after the step. It is always an Array. Each
// entry of that array is an element that reflects the returns from the iterated
// step.
//
// An step that only returns one element will produce an array of such elements
// An step that produces multiple elements will produce an array where each
// element is a hash
func (it *iterator) Returns() []px.Parameter {
	returns := it.Producer().Returns()
	var vt px.Type
	if len(returns) == 1 {
		vt = returns[0].Type()
	} else {
		se := make([]*types.StructElement, len(returns))
		for i, p := range returns {
			se[i] = types.NewStructElement(types.WrapString(p.Name()), p.Type())
		}
		vt = types.NewStructType(se)
	}
	return []px.Parameter{
		px.NewParameter(it.resultName, types.NewArrayType(vt, nil), nil, false)}
}

func (it *iterator) Variables() []px.Parameter {
	return it.variables
}

func (it *iterator) iterate(ctx px.Context, vars px.OrderedMap, start, end int64, iterFunc func(int) px.OrderedMap) px.OrderedMap {

	done := make(chan bool)
	count := end - start
	numWorkers := int(count)
	if count > maxParallel {
		numWorkers = maxParallel
	}

	els := make([]px.Value, count)
	jobs := make(chan int64)
	for i := 0; i < numWorkers; i++ {
		px.Fork(ctx, func(fc px.Context) {
			for ix := range jobs {
				func() {
					defer func() {
						if atomic.AddInt64(&count, -1) <= 0 {
							close(jobs)
							done <- true
						}
					}()
					p := it.Producer()
					parameters := iterFunc(int(ix))
					ip := p.WithIndex(int(ix))
					result := ip.Run(fc, parameters)
					v := px.Undef
					if len(p.Returns()) == 1 {
						if result.Len() > 0 {
							v = result.At(0).(px.MapEntry).Value()
						}
					} else {
						v = result
					}
					els[ix-start] = v
				}()
			}
		})
	}

	for i := start; i < end; i++ {
		jobs <- i
	}
	<-done
	return px.SingletonMap(it.resultName, types.WrapValues(els))
}

func resolveParameters(c px.Context, it api.Iterator, parameters px.OrderedMap) (px.Value, px.OrderedMap) {
	// Resolve the parameters that acts as parameters to the iteration.
	vars := make([]*types.HashEntry, 0, len(it.Parameters()))

	// Strip parameters intended for the iterator from the list intended for the step that will be called by the iterator
	for _, ap := range it.Producer().Parameters() {
		if ev, ok := parameters.GetEntry(ap.Name()); ok {
			vars = append(vars, ev.(*types.HashEntry))
		}
	}
	return types.ResolveDeferred(c, it.Over(), c.Scope()), types.WrapHash(vars)
}

func Validate(it api.Iterator) {
	a := it.Producer()

	// Ensure that parameters contains returns produced by the iterator
	is := a.Parameters()
	vs := it.Variables()
nextVar:
	for _, v := range vs {
		for _, i := range is {
			if i.Name() == v.Name() {
				continue nextVar
			}
		}
		panic(px.Error(IterationStepWrongParameters, issue.H{`iterator`: it}))
	}
}

func assertInt(t api.Iterator, arg px.Value, name string) int64 {
	iv, ok := arg.(px.Integer)
	if !ok {
		panic(px.Error(IterationParameterWrongType, issue.H{
			`iterator`: t, `parameter`: name, `expected`: `Integer`, `actual`: arg.PType()}))
	}
	return iv.Int()
}

func assertRange(t api.Iterator, arg px.Value) (int64, int64) {
	a, ok := arg.(*types.Array)
	if !(ok && a.Len() == 2) {
		panic(px.Error(IterationParameterWrongType, issue.H{
			`iterator`: t, `parameter`: `over`, `expected`: `Array`, `actual`: arg.PType()}))
	}
	return assertInt(t, a.At(0), `over[0]`), assertInt(t, a.At(1), `over[1]`)
}

func assertList(t api.Iterator, arg px.Value) px.List {
	if a, ok := arg.(*types.Array); ok {
		return a
	}
	panic(px.Error(IterationParameterWrongType, issue.H{
		`iterator`: t, `parameter`: `over`, `expected`: `Array`, `actual`: arg.PType()}))
}

func assertMap(t api.Iterator, arg px.Value) px.OrderedMap {
	if h, ok := arg.(px.OrderedMap); ok {
		return h
	}
	panic(px.Error(IterationParameterWrongType, issue.H{
		`iterator`: t, `parameter`: `over`, `expected`: `Hash`, `actual`: arg.PType()}))
}

func iterLabel(it api.Iterator) string {
	return fmt.Sprintf(`%s %s iteration`, it.Style(), StepLabel(it))
}

type each struct {
	iterator
}

func NewEach(step api.Step, name string, over px.Value, variables []px.Parameter) api.Iterator {
	it := &each{iterator{step, over, variables, name}}
	Validate(it)
	return it
}

func (t *each) Label() string {
	return iterLabel(t)
}

func (t *each) IterationStyle() wf.IterationStyle {
	return wf.IterationStyleEach
}

func (t *each) Run(ctx px.Context, parameters px.OrderedMap) px.OrderedMap {
	over, vars := resolveParameters(ctx, t, parameters)
	list := assertList(t, over)
	return t.iterate(ctx, vars, 0, int64(list.Len()), func(ix int) px.OrderedMap {
		vs := t.Variables()
		nv := len(vs)
		el := list.At(ix)
		switch nv {
		case 0:
			// Do nothing
		case 1:
			parameters = parameters.Merge(px.SingletonMap(vs[0].Name(), el))
		default:
			es := make([]*types.HashEntry, 0, len(vs))
			switch el := el.(type) {
			case *types.HashEntry:
				// Map key and value to first two positions
				es = append(es, types.WrapHashEntry2(vs[0].Name(), el.Key()), types.WrapHashEntry2(vs[1].Name(), el.Value()))
			case *types.Array:
				// Map as many as possible by index
				el.EachWithIndex(func(e px.Value, i int) {
					if i < nv {
						es = append(es, types.WrapHashEntry2(vs[i].Name(), e))
					}
				})
			case *types.Hash:
				// Map as many as possible by name
				for _, p := range vs {
					if v, ok := el.Get4(p.Name()); ok {
						es = append(es, types.WrapHashEntry2(p.Name(), v))
					}
				}
			case px.PuppetObject:
				// Map as many as possible by name
				pt := el.PType().(px.ObjectType)
				for _, p := range vs {
					if v, ok := pt.Member(p.Name()); ok {
						if a, ok := v.(px.Attribute); ok {
							es = append(es, types.WrapHashEntry2(p.Name(), a.Get(el)))
						}
					}
				}
			default:
				es = append(es, types.WrapHashEntry2(vs[0].Name(), el))
			}

			if len(es) > 0 {
				parameters = parameters.Merge(types.WrapHash(es))
			}
		}
		return parameters
	})
}

type eachPair struct {
	iterator
}

func NewEachPair(step api.Step, name string, over px.Value, variables []px.Parameter) api.Iterator {
	it := &eachPair{iterator{step, over, variables, name}}
	Validate(it)
	return it
}

func (t *eachPair) Label() string {
	return iterLabel(t)
}

func (t *eachPair) IterationStyle() wf.IterationStyle {
	return wf.IterationStyleEachPair
}

func (t *eachPair) Run(ctx px.Context, parameters px.OrderedMap) px.OrderedMap {
	over, vars := resolveParameters(ctx, t, parameters)
	mp := assertMap(t, over)
	p0 := t.Variables()[0].Name()
	p1 := t.Variables()[1].Name()
	return t.iterate(ctx, vars, 0, int64(mp.Len()), func(ix int) px.OrderedMap {
		entry := mp.At(ix).(px.MapEntry)
		ke := types.WrapHashEntry2(p0, entry.Key())
		ve := types.WrapHashEntry2(p1, entry.Value())
		return vars.Merge(types.WrapHash([]*types.HashEntry{ke, ve}))
	})
}

type times struct {
	iterator
}

func NewTimes(step api.Step, name string, over px.Value, variables []px.Parameter) api.Iterator {
	it := &times{iterator{step, over, variables, name}}
	Validate(it)
	return it
}

func (t *times) Label() string {
	return iterLabel(t)
}

func (t *times) IterationStyle() wf.IterationStyle {
	return wf.IterationStyleTimes
}

func (t *times) Run(ctx px.Context, parameters px.OrderedMap) px.OrderedMap {
	over, vars := resolveParameters(ctx, t, parameters)
	return t.iterate(ctx, vars, 0, assertInt(t, over, `over`), func(ix int) px.OrderedMap {
		vs := t.Variables()
		if len(vs) > 0 {
			parameters = parameters.Merge(px.SingletonMap(t.Variables()[0].Name(), types.WrapInteger(int64(ix))))
		}
		return parameters
	})
}

type itRange struct {
	iterator
}

func NewRange(step api.Step, name string, over px.Value, variables []px.Parameter) api.Iterator {
	it := &itRange{iterator{step, over, variables, name}}
	Validate(it)
	return it
}

func (t *itRange) Label() string {
	return iterLabel(t)
}

func (t *itRange) IterationStyle() wf.IterationStyle {
	return wf.IterationStyleRange
}

func (t *itRange) Run(ctx px.Context, parameters px.OrderedMap) px.OrderedMap {
	over, vars := resolveParameters(ctx, t, parameters)
	from, to := assertRange(t, over)
	return t.iterate(ctx, vars, from, to, func(ix int) px.OrderedMap {
		vs := t.Variables()
		if len(vs) > 0 {
			parameters = parameters.Merge(px.SingletonMap(t.Variables()[0].Name(), types.WrapInteger(int64(ix))))
		}
		return parameters
	})
}
