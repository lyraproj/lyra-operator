package service

import (
	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/annotation"
	"github.com/lyraproj/servicesdk/grpc"
	"github.com/lyraproj/servicesdk/service"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/wf"
	"github.com/lyraproj/wfe/api"
)

func StartEra(c px.Context) {
	getIdentity(c).bumpEra(c)
}

// SweepAndGC performs a sweep of the Identity store, retrieves all garbage, and
// then tells the handler for each garbage entry to delete the resource. The entry
// is then purged from the Identity store
func SweepAndGC(c px.Context, prefix string) {
	identity := getIdentity(c)
	log := hclog.Default()
	log.Debug("GC Sweep", "prefix", prefix)
	identity.sweep(c, prefix)
	log.Debug("GC Collect garbage", "prefix", prefix)
	gl := identity.garbage(c, prefix)
	ng := gl.Len()
	log.Debug("GC Collect garbage", "prefix", prefix, "count", ng)
	rs := make([]px.List, ng)

	// Store in reverse order
	ng--
	gl.EachWithIndex(func(t px.Value, i int) {
		rs[ng-i] = t.(px.List)
	})

	for _, l := range rs {
		uri := types.ParseURI(l.At(0).String())
		hid := uri.Query().Get(`hid`)
		handlerDef := GetHandler(c, px.NewTypedName(px.NsHandler, hid))
		handler := GetService(c, handlerDef.ServiceId())

		extId := l.At(1)
		log.Debug("GC delete", "prefix", prefix, "intId", uri.String(), "extId", extId)
		handler.Invoke(c, handlerDef.Identifier().Name(), `delete`, extId)
		identity.purgeExternal(c, extId)
	}
}

func readOrNotFound(c px.Context, handler serviceapi.Service, hn string, extId px.Value, identity *identity) px.Value {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(issue.Reported); ok && e.Code() == grpc.InvocationError {
				if e.Argument(`code`) == service.NotFound {
					// Not found by remote. Purge the extId and return nil.
					hclog.Default().Debug("Removing obsolete extId from Identity service", "extId", extId)
					identity.purgeExternal(c, extId)
					return
				}
			}
			panic(r)
		}
	}()

	hclog.Default().Debug("Read state", "extId", extId)
	return handler.Invoke(c, hn, `read`, extId)
}
func ApplyState(c px.Context, resource api.Resource, parameters px.OrderedMap) px.OrderedMap {
	ac := StepContext(c)
	op := GetOperation(ac)

	log := hclog.Default()
	handlerDef := GetHandler(c, resource.HandlerId())
	crd := GetProperty(handlerDef, `interface`, types.NewTypeType(types.DefaultObjectType())).(px.ObjectType)
	identity := getIdentity(c)
	handler := GetService(c, handlerDef.ServiceId())

	intId := types.WrapString(resource.Identifier())
	extId := resource.ExtId()
	explicitExtId := extId != nil
	if !explicitExtId {
		// external id must exist in order to do a read or delete
		extId = identity.getExternal(c, intId, op == wf.Read || op == wf.Delete)
		log.Debug("GetExternal", "intId", intId, "extId", extId)
	}

	var result px.PuppetObject
	hn := handlerDef.Identifier().Name()
	switch op {
	case wf.Read:
		if extId == nil {
			return px.EmptyMap
		}
		rt := readOrNotFound(c, handler, hn, extId, identity)
		if rt == nil {
			return px.EmptyMap
		}
		result = px.AssertInstance(handlerDef.Label, resource.Type(), rt).(px.PuppetObject)

	case wf.Upsert:
		if explicitExtId {
			// An explicit externalId is for resources not managed by us. Only possible action
			// here is a read
			rt := readOrNotFound(c, handler, hn, extId, identity)
			if rt == nil {
				// False positive from the Identity service
				return px.EmptyMap
			}
			result = px.AssertInstance(handlerDef.Label, resource.Type(), rt).(px.PuppetObject)
			break
		}

		desiredState := GetService(c, resource.ServiceId()).State(c, resource.Name(), parameters)
		if extId != nil {
			// Read current state and check if an update is needed
			rt := readOrNotFound(c, handler, hn, extId, identity)
			if rt == nil {
				// False positive from the Identity service
				extId = nil
			} else {
				result = px.AssertInstance(handlerDef.Label, resource.Type(), rt).(px.PuppetObject)
			}
		}

		if extId == nil {
			// Nothing exists yet. Create a new instance
			log.Debug("Create state", "intId", intId)
			rt := handler.Invoke(c, hn, `create`, desiredState).(px.List)
			result = px.AssertInstance(handlerDef.Label, resource.Type(), rt.At(0)).(px.PuppetObject)
			extId = rt.At(1)
			log.Debug("Associate state", "intId", intId, "extId", extId)
			identity.associate(c, intId, extId)
			break
		}

		var updateNeeded, recreateNeeded bool
		if a, ok := resource.Type().Annotations(c).Get(annotation.ResourceType); ok {
			ra := a.(annotation.Resource)
			updateNeeded, recreateNeeded = ra.Changed(desiredState, result)
		} else {
			updateNeeded = !desiredState.Equals(result, nil)
			recreateNeeded = false
		}

		if updateNeeded {
			if !recreateNeeded {
				// Update existing content. If an update method exists, call it. If not, then fall back
				// to delete + create
				if _, ok := crd.Member(`update`); ok {
					log.Debug("Update state", "extId", extId)
					result = px.AssertInstance(handlerDef.Label, resource.Type(), handler.Invoke(c, hn, `update`, extId, desiredState)).(px.PuppetObject)
					break
				}
			}

			// Rely on that deletion happens by means of GC at end of run
			log.Debug("Remove external", "extId", extId)
			identity.removeExternal(c, extId)

			log.Debug("Create state", "intId", intId)
			rt := handler.Invoke(c, hn, `create`, desiredState)
			rl := rt.(px.List)
			result = px.AssertInstance(handlerDef.Label, resource.Type(), rl.At(0)).(px.PuppetObject)
			extId = rl.At(1)
			log.Debug("Associate state", "intId", intId, "extId", extId)
			identity.associate(c, intId, extId)
		}
	default:
		panic(px.Error(wf.IllegalOperation, issue.H{`operation`: op}))
	}

	switch op {
	case wf.Read, wf.Upsert:
		returns := resource.Returns()
		entries := make([]*types.HashEntry, len(returns))
		for i, o := range returns {
			entries[i] = getValue(o, resource, result)
		}
		return types.WrapHash(entries)
	}
	return px.EmptyMap
}

func getValue(p px.Parameter, r api.Resource, o px.PuppetObject) *types.HashEntry {
	n := p.Name()
	a := n
	if p.HasValue() {
		v := p.Value()
		if a, ok := v.(*types.Array); ok {
			// Build hash from multiple attributes
			entries := make([]*types.HashEntry, a.Len())
			a.EachWithIndex(func(e px.Value, i int) {
				a := e.String()
				if v, ok := o.Get(a); ok {
					entries[i] = types.WrapHashEntry(e, v)
				} else {
					panic(px.Error(api.NoSuchAttribute, issue.H{`step`: r, `name`: a}))
				}
			})
			return types.WrapHashEntry2(n, types.WrapHash(entries))
		}

		if s, ok := v.(px.StringValue); ok {
			a = s.String()
		}
	}
	if v, ok := o.Get(a); ok {
		return types.WrapHashEntry2(n, v)
	}
	panic(px.Error(api.NoSuchAttribute, issue.H{`step`: r, `name`: a}))
}
