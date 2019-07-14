package serialization

import (
	"bytes"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type dsContext struct {
	types.BasicCollector
	allowUnresolved bool
	context         px.Context
	newTypes        []px.Type
	value           px.Value
	converted       map[px.Value]px.Value
}

// NewDeserializer creates a new Collector that consumes input and creates a RichData Value
func NewDeserializer(ctx px.Context, options px.OrderedMap) px.Collector {
	ds := &dsContext{
		context:         ctx,
		newTypes:        make([]px.Type, 0, 11),
		converted:       make(map[px.Value]px.Value, 11),
		allowUnresolved: options.Get5(`allow_unresolved`, types.BooleanFalse).(px.Boolean).Bool()}
	ds.Init()
	return ds
}

func (ds *dsContext) Value() px.Value {
	if ds.value == nil {
		ds.value = ds.convert(ds.BasicCollector.Value())
		px.AddTypes(ds.context, ds.newTypes...)
	}
	return ds.value
}

func (ds *dsContext) convert(value px.Value) px.Value {
	if cv, ok := ds.converted[value]; ok {
		return cv
	}

	if hash, ok := value.(*types.Hash); ok {
		if hash.AllKeysAreStrings() {
			if pcoreType, ok := hash.Get4(PcoreTypeKey); ok {
				switch pcoreType.String() {
				case PcoreTypeHash:
					return ds.convertHash(hash)
				case PcoreTypeSensitive:
					return ds.convertSensitive(hash)
				case PcoreTypeDefault:
					return types.WrapDefault()
				default:
					v := ds.convertOther(hash, pcoreType)
					switch v.(type) {
					case px.ObjectType, px.TypeSet, *types.TypeAliasType:
						// Ensure that type is made known to current loader
						rt := v.(px.ResolvableType)
						n := rt.Name()
						if n == `` {
							// Anonymous type. Just resolve.
							return rt.Resolve(ds.context)
						}
						// Duplicates can be found here if serialization was made with dedupLevel NoDedup
						for _, nt := range ds.newTypes {
							if n == nt.Name() {
								return nt
							}
						}
						tn := px.NewTypedName(px.NsType, n)
						if lt, ok := px.Load(ds.context, tn); ok {
							t := rt.Resolve(ds.context)
							if t.Equals(lt, nil) {
								return lt.(px.Value)
							}
							ob := bytes.NewBufferString(``)
							lt.(px.Type).ToString(ob, px.PrettyExpanded, nil)
							nb := bytes.NewBufferString(``)
							t.(px.Type).ToString(nb, px.PrettyExpanded, nil)
							panic(px.Error(px.AttemptToRedefineType, issue.H{`name`: tn, `old`: ob.String(), `new`: nb.String()}))
						}
						ds.newTypes = append(ds.newTypes, v.(px.Type))
					}
					return v
				}
			}
		}

		return types.BuildHash(hash.Len(), func(h *types.Hash, entries []*types.HashEntry) []*types.HashEntry {
			ds.converted[value] = h
			hash.EachPair(func(k, v px.Value) {
				entries = append(entries, types.WrapHashEntry(ds.convert(k), ds.convert(v)))
			})
			return entries
		})
	}

	if array, ok := value.(*types.Array); ok {
		return types.BuildArray(array.Len(), func(a *types.Array, elements []px.Value) []px.Value {
			ds.converted[value] = a
			array.Each(func(v px.Value) { elements = append(elements, ds.convert(v)) })
			return elements
		})
	}
	return value
}

func (ds *dsContext) convertHash(hv px.OrderedMap) px.Value {
	value := hv.Get5(PcoreValueKey, px.EmptyArray).(px.List)
	return types.BuildHash(value.Len(), func(hash *types.Hash, entries []*types.HashEntry) []*types.HashEntry {
		ds.converted[hv] = hash
		for idx := 0; idx < value.Len(); idx += 2 {
			entries = append(entries, types.WrapHashEntry(ds.convert(value.At(idx)), ds.convert(value.At(idx+1))))
		}
		return entries
	})
}

func (ds *dsContext) convertSensitive(hash px.OrderedMap) px.Value {
	cv := types.WrapSensitive(ds.convert(hash.Get5(PcoreValueKey, px.Undef)))
	ds.converted[hash] = cv
	return cv
}

func (ds *dsContext) convertOther(hash px.OrderedMap, typeValue px.Value) px.Value {
	value := hash.Get6(PcoreValueKey, func() px.Value {
		return hash.RejectPairs(func(k, v px.Value) bool {
			if s, ok := k.(px.StringValue); ok {
				return s.String() == PcoreTypeKey
			}
			return false
		})
	})
	if typeHash, ok := typeValue.(*types.Hash); ok {
		typ := ds.convert(typeHash)
		if _, ok := typ.(*types.Hash); ok {
			if !ds.allowUnresolved {
				panic(px.Error(px.UnableToDeserializeType, issue.H{`hash`: typ.String()}))
			}
			return hash
		}
		return ds.pcoreTypeHashToValue(typ.(px.Type), hash, value)
	}
	typ := ds.context.ParseTypeValue(typeValue)
	if tr, ok := typ.(*types.TypeReferenceType); ok {
		if !ds.allowUnresolved {
			panic(px.Error(px.UnresolvedType, issue.H{`typeString`: tr.String()}))
		}
		return hash
	}
	return ds.pcoreTypeHashToValue(typ.(px.Type), hash, value)
}

func (ds *dsContext) pcoreTypeHashToValue(typ px.Type, key, value px.Value) px.Value {
	var ov px.Value
	if hash, ok := value.(*types.Hash); ok {
		args := ds.convert(hash)

		if ov, ok = ds.allocate(typ); ok {
			ds.converted[key] = ov
			ov.(px.Object).InitFromHash(ds.context, args.(*types.Hash))
			return ov
		}

		if ot, ok := typ.(px.ObjectType); ok {
			if ot.HasHashConstructor() {
				ov = px.New(ds.context, typ, args)
			} else {
				ov = px.New(ds.context, typ, ot.AttributesInfo().PositionalFromHash(args.(*types.Hash))...)
			}
		} else {
			ov = px.New(ds.context, typ, args)
		}
	} else {
		if str, ok := value.(px.StringValue); ok {
			ov = px.New(ds.context, typ, str)
		} else {
			panic(px.Error(px.UnableToDeserializeValue, issue.H{`type`: typ.Name(), `arg_type`: value.PType().Name()}))
		}
	}
	ds.converted[key] = ov
	return ov
}

func (ds *dsContext) allocate(typ px.Type) (px.Object, bool) {
	if allocator, ok := px.Load(ds.context, px.NewTypedName(px.NsAllocator, typ.Name())); ok {
		return allocator.(px.Lambda).Call(nil, nil).(px.Object), true
	}
	if ot, ok := typ.(px.ObjectType); ok && ot.Name() == `Pcore::ObjectType` {
		return types.AllocObjectType(), true
	}
	return nil, false
}
