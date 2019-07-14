package wfe

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/wf"
	"github.com/lyraproj/wfe/api"
	"github.com/lyraproj/wfe/service"
)

type Step struct {
	serviceId  px.TypedName
	name       string
	when       wf.Condition
	parameters []px.Parameter
	returns    []px.Parameter
	index      int
}

func CreateStep(c px.Context, def serviceapi.Definition) api.Step {
	hclog.Default().Debug(`creating step`, `style`, service.GetStringProperty(def, `style`))

	switch service.GetStringProperty(def, `style`) {
	case `stateHandler`:
		return StateHandler(def)
	case `iterator`:
		return Iterator(c, def)
	case `resource`:
		return Resource(c, def)
	case `workflow`:
		return Workflow(c, def)
	case `action`:
		return Action(def)
	case `reference`:
		return Reference(c, def)
	}
	return nil
}

func (a *Step) GetService(c px.Context) serviceapi.Service {
	return service.GetService(c, a.serviceId)
}

func (a *Step) ServiceId() px.TypedName {
	return a.serviceId
}

func StepLabel(a api.Step) string {
	return fmt.Sprintf(`%s '%s'`, a.Style(), a.Name())
}

func StepId(a api.Step) string {
	b := bytes.NewBufferString(`lyra://puppet.com`)
	for _, s := range strings.Split(a.Name(), `::`) {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(s))
	}
	vs := a.IdParams()
	if len(vs) > 0 {
		b.WriteByte('?')
		b.WriteString(vs.Encode())
	}
	return b.String()
}

func (a *Step) When() wf.Condition {
	return a.when
}

func (a *Step) Name() string {
	return a.name
}

func (a *Step) Parameters() []px.Parameter {
	return a.parameters
}

func (a *Step) Returns() []px.Parameter {
	return a.returns
}

func (a *Step) Init(def serviceapi.Definition) {
	a.index = -1
	a.serviceId = def.ServiceId()
	a.name = def.Identifier().Name()
	props := def.Properties()
	a.parameters = getParameters(`parameters`, props)
	a.returns = getParameters(`returns`, props)
	if wh, ok := props.Get4(`when`); ok {
		a.when = wh.(wf.Condition)
	} else {
		a.when = wf.Always
	}
}

func getParameters(key string, props px.OrderedMap) []px.Parameter {
	if parameters, ok := props.Get4(key); ok {
		ia := parameters.(px.List)
		is := make([]px.Parameter, ia.Len())
		ia.EachWithIndex(func(iv px.Value, idx int) { is[idx] = iv.(px.Parameter) })
		return is
	}
	return []px.Parameter{}
}

func (a *Step) IdParams() url.Values {
	if a.index >= 0 {
		return url.Values{`index`: {strconv.Itoa(a.index)}}
	}
	return url.Values{}
}

func ResolveParameters(ctx px.Context, a api.Step, parameters px.OrderedMap, p px.Parameter) px.Value {
	if !p.HasValue() {
		if v, ok := parameters.Get4(p.Name()); ok {
			return v
		}
		panic(px.Error(ParameterUnresolved, issue.H{`step`: a, `parameter`: p.Name()}))
	}
	return types.ResolveDeferred(ctx, p.Value(), parameters)
}

// setIndex must only be called after a direct cloning operation on the instance, i.e. from WithIndex()
func (a *Step) setIndex(index int) {
	a.index = index
}
