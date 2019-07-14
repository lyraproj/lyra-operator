package wfe

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/wf"
	"github.com/lyraproj/wfe/api"
	"github.com/lyraproj/wfe/service"
)

type stateHandler struct {
	Step
	typ px.ObjectType
}

func StateHandler(def serviceapi.Definition) api.Step {
	a := &stateHandler{}
	a.Init(def)
	return a
}

func (a *stateHandler) Init(def serviceapi.Definition) {
	a.Step.Init(def)
	// TODO: Type validation. The typ must be an ObjectType implementing read, upsert, and delete.
	a.typ = service.GetProperty(def, `interface`, types.NewTypeType(types.DefaultObjectType())).(px.ObjectType)
}

func (a *stateHandler) Run(c px.Context, parameters px.OrderedMap) px.OrderedMap {
	ac := service.StepContext(c)
	op := service.GetOperation(ac)
	invokable := a.GetService(c)

	switch op {
	case wf.Read:
		return invokable.Invoke(c, a.name, `read`, parameters).(px.OrderedMap)

	case wf.Upsert:
		return invokable.Invoke(c, a.name, `upsert`, parameters).(px.OrderedMap)

	case wf.Delete:
		invokable.Invoke(c, a.name, `delete`, parameters)
		return px.EmptyMap
	default:
		panic(px.Error(wf.IllegalOperation, issue.H{`operation`: op}))
	}
}

func (a *stateHandler) Identifier() string {
	return StepId(a)
}

func (a *stateHandler) Label() string {
	return StepLabel(a)
}

func (a *stateHandler) Style() string {
	return `stateHandler`
}

func (a *stateHandler) WithIndex(index int) api.Step {
	ac := stateHandler{}
	ac = *a // Copy by value
	ac.setIndex(index)
	return &ac
}
