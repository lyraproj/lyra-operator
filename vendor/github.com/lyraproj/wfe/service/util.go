package service

import (
	"fmt"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/wf"
	"github.com/lyraproj/wfe/api"
)

const StepContextKey = `step::context`

func StepContext(c px.Context) px.OrderedMap {
	if ac, ok := c.Get(StepContextKey); ok {
		return px.AssertInstance(`invalid step context`, types.DefaultHashType(), ac.(px.Value)).(px.OrderedMap)
	}
	panic(px.Error(api.NoStepContext, issue.NoArgs))
}

func GetOperation(ac px.OrderedMap) wf.Operation {
	if op, ok := ac.Get4(`operation`); ok {
		return wf.Operation(op.(px.Integer).Int())
	}
	return wf.Read
}

func GetService(c px.Context, serviceId px.TypedName) serviceapi.Service {
	if serviceId.Namespace() == px.NsService {
		if sm, ok := px.Load(c, serviceId); ok {
			return sm.(serviceapi.Service)
		}
	}
	panic(px.Error(api.UnableToLoadRequired, issue.H{`namespace`: string(px.NsService), `name`: serviceId.String()}))
}

func GetDefinition(c px.Context, definitionId px.TypedName) serviceapi.Definition {
	if definitionId.Namespace() == px.NsDefinition {
		if sm, ok := px.Load(c, definitionId); ok {
			return sm.(serviceapi.Definition)
		}
	}
	panic(px.Error(api.UnableToLoadRequired, issue.H{`namespace`: string(px.NsDefinition), `name`: definitionId.String()}))
}

func GetHandler(c px.Context, handlerId px.TypedName) serviceapi.Definition {
	if handlerId.Namespace() == px.NsHandler {
		if sm, ok := px.Load(c, handlerId); ok {
			return sm.(serviceapi.Definition)
		}
	}
	panic(px.Error(api.UnableToLoadRequired, issue.H{`namespace`: string(px.NsHandler), `name`: handlerId.String()}))
}

func GetStringProperty(def serviceapi.Definition, key string) string {
	return GetProperty(def, key, types.DefaultStringType()).String()
}

func GetProperty(def serviceapi.Definition, key string, typ px.Type) px.Value {
	if prop, ok := def.Properties().Get4(key); ok {
		return px.AssertInstance(func() string {
			return fmt.Sprintf(`%s %s, property %s`, def.ServiceId(), def.Identifier(), key)
		}, typ, prop)
	}
	panic(px.Error(api.MissingRequiredProperty, issue.H{`service`: def.ServiceId(), `definition`: def.Identifier(), `key`: key}))
}

func GetOptionalProperty(def serviceapi.Definition, key string, typ px.Type) (px.Value, bool) {
	if prop, ok := def.Properties().Get4(key); ok {
		return px.AssertInstance(func() string {
			return fmt.Sprintf(`%s %s, property %s`, def.ServiceId(), def.Identifier(), key)
		}, typ, prop), true
	}
	return nil, false
}

func LeafName(name string) string {
	names := strings.Split(name, `::`)
	return names[len(names)-1]
}
