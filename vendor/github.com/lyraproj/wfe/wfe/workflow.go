package wfe

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/wfe/api"
	"github.com/lyraproj/wfe/service"
)

type workflow struct {
	Step

	steps []api.Step
}

var DefinitionListType = types.NewArrayType(serviceapi.DefinitionMetaType, nil)

func (w *workflow) Run(ctx px.Context, parameters px.OrderedMap) px.OrderedMap {
	wf := NewWorkflowEngine(w)
	wf.Validate()
	return wf.Run(ctx, parameters)
}

func (w *workflow) Identifier() string {
	return StepId(w)
}

func (w *workflow) Label() string {
	return StepLabel(w)
}

func (w *workflow) Style() string {
	return `workflow`
}

func (w *workflow) WithIndex(index int) api.Step {
	wc := *w // Copy by value
	wc.setIndex(index)
	return &wc
}

func Workflow(c px.Context, def serviceapi.Definition) api.Workflow {
	wf := &workflow{}
	wf.Init(def)
	steps := service.GetProperty(def, `steps`, DefinitionListType).(px.List)
	as := make([]api.Step, steps.Len())
	steps.EachWithIndex(func(v px.Value, i int) { as[i] = CreateStep(c, v.(serviceapi.Definition)) })
	wf.steps = as
	return wf
}

func (w *workflow) Steps() []api.Step {
	return w.steps
}
