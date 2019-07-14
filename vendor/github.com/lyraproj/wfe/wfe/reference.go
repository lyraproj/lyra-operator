package wfe

import (
	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/wf"
	"github.com/lyraproj/wfe/api"
	"github.com/lyraproj/wfe/service"
)

type reference struct {
	Step
	ra api.Step
}

func Reference(c px.Context, def serviceapi.Definition) api.Step {
	r := &reference{}
	r.Init(def)
	reference := service.GetStringProperty(def, `reference`)
	hclog.Default().Debug(`resolving activity reference`, `name`, r.name, `reference`, reference)
	r.ra = CreateStep(c, service.GetDefinition(c, px.NewTypedName(px.NsDefinition, reference)))
	return r
}

func (r *reference) Identifier() string {
	return StepId(r)
}

func (r *reference) Parameters() []px.Parameter {
	var input []px.Parameter
	if len(r.parameters) == 0 {
		input = r.ra.Parameters()
	} else {
		// Return a copy with the alias stripped of. Input parameters with values (alias is a value)
		// doesn't pick up a new value from the workflow.
		input = make([]px.Parameter, len(r.parameters))
		for i, p := range r.parameters {
			input[i] = px.NewParameter(p.Name(), p.Type(), nil, false)
		}
	}
	return input
}

func (r *reference) Returns() []px.Parameter {
	output := r.returns
	if len(output) == 0 {
		output = r.ra.Returns()
	}
	return output
}

func (r *reference) When() wf.Condition {
	when := r.when
	if when == nil {
		when = r.ra.When()
	} else {
		if r.ra.When() != nil {
			when = wf.And([]wf.Condition{when, r.ra.When()})
		}
	}
	return when
}

func (r *reference) Run(ctx px.Context, input px.OrderedMap) px.OrderedMap {
	return r.mapOutput(r.ra.Run(ctx, r.mapInput(input)))
}

func (r *reference) Label() string {
	return StepLabel(r)
}

func (r *reference) Style() string {
	return `reference`
}

func (r *reference) mapInput(input px.OrderedMap) px.OrderedMap {
	ips := r.parameters
	if len(ips) == 0 {
		ips = r.ra.Parameters()
		if len(ips) == 0 {
			return input
		}
	}
	return input.MapEntries(func(entry px.MapEntry) px.MapEntry {
		key := entry.Key()
		kn := key.String()
		for _, p := range ips {
			if p.Name() == kn {
				if p.HasValue() {
					if alias, ok := p.Value().(px.StringValue); ok {
						entry = types.WrapHashEntry(alias, entry.Value())
					}
				}
				break
			}
		}
		return entry
	})
}

func (r *reference) mapOutput(output px.OrderedMap) px.OrderedMap {
	ops := r.Returns()
	if len(ops) == 0 {
		return output
	}
	return output.MapEntries(func(entry px.MapEntry) px.MapEntry {
		key := entry.Key()
		for _, p := range ops {
			if p.HasValue() {
				if alias, ok := p.Value().(px.StringValue); ok && alias.Equals(key, nil) {
					entry = types.WrapHashEntry2(p.Name(), entry.Value())
					break
				}
			}
		}
		return entry
	})
}

func (r *reference) WithIndex(index int) api.Step {
	rc := *r // Copy by value
	rc.setIndex(index)
	return &rc
}
