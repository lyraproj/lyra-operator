package wfe

import (
	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/wfe/api"
)

type action struct {
	Step
	api px.ObjectType
}

/* TODO: Add type check using expectedType
var ioType = types.NewHashType(types.DefaultStringType(), types.DefaultRichDataType(), nil)
var expectedType = types.NewCallableType(
	types.NewTupleType([]px.Type{ioType}, nil), ioType, nil)
*/

func Action(def serviceapi.Definition) api.Step {
	a := &action{}
	a.Init(def)
	return a
}

func (s *action) Init(d serviceapi.Definition) {
	s.Step.Init(d)
	if i, ok := d.Properties().Get4(`interface`); ok {
		s.api = i.(px.ObjectType)
	}
}

func (s *action) Run(ctx px.Context, parameters px.OrderedMap) px.OrderedMap {
	service := s.GetService(ctx)
	hclog.Default().Debug(`executing action`, `name`, s.name)
	result := service.Invoke(ctx, s.Name(), `do`, parameters)
	if m, ok := result.(px.OrderedMap); ok {
		return m
	}
	panic(result.String())
}

func (s *action) Label() string {
	return StepLabel(s)
}

func (a *action) Identifier() string {
	return StepId(a)
}

func (s *action) Style() string {
	return `action`
}

func (a *action) WithIndex(index int) api.Step {
	ac := *a // Copy by value
	ac.setIndex(index)
	return &ac
}
