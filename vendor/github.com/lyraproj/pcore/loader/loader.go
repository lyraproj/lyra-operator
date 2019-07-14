package loader

import (
	"bytes"
	"sort"
	"sync"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type (
	loaderEntry struct {
		value  interface{}
		origin issue.Location
	}

	basicLoader struct {
		lock         sync.RWMutex
		namedEntries map[string]px.LoaderEntry
	}

	parentedLoader struct {
		basicLoader
		parent px.Loader
	}

	typeSetLoader struct {
		parentedLoader
		typeSet px.TypeSet
	}
)

var StaticLoader = &basicLoader{namedEntries: make(map[string]px.LoaderEntry, 64)}

func init() {
	sh := StaticLoader.namedEntries
	types.EachCoreType(func(t px.Type) {
		sh[px.NewTypedName(px.NsType, t.Name()).MapKey()] = &loaderEntry{t, nil}
	})

	px.StaticLoader = func() px.Loader {
		return StaticLoader
	}

	px.NewParentedLoader = func(parent px.Loader) px.DefiningLoader {
		return &parentedLoader{basicLoader{namedEntries: make(map[string]px.LoaderEntry, 64)}, parent}
	}

	px.NewTypeSetLoader = func(parent px.Loader, typeSet px.Type) px.TypeSetLoader {
		return &typeSetLoader{parentedLoader{basicLoader{namedEntries: make(map[string]px.LoaderEntry, 64)}, parent}, typeSet.(px.TypeSet)}
	}

	px.NewLoaderEntry = func(value interface{}, origin issue.Location) px.LoaderEntry {
		return &loaderEntry{value, origin}
	}

	px.Load = load
}

func (e *loaderEntry) Origin() issue.Location {
	return e.origin
}

func (e *loaderEntry) Value() interface{} {
	return e.value
}

func load(c px.Context, name px.TypedName) (interface{}, bool) {
	l := c.Loader()
	if name.Authority() != l.NameAuthority() {
		return nil, false
	}
	entry := l.LoadEntry(c, name)
	if entry == nil {
		if dl, ok := l.(px.DefiningLoader); ok {
			dl.SetEntry(name, &loaderEntry{nil, nil})
		}
		return nil, false
	}
	if entry.Value() == nil {
		return nil, false
	}
	return entry.Value(), true
}

func (l *basicLoader) Discover(c px.Context, predicate func(tn px.TypedName) bool) []px.TypedName {
	found := make([]px.TypedName, 0)
	for k := range l.namedEntries {
		tn := px.TypedNameFromMapKey(k)
		if predicate(tn) {
			found = append(found, tn)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].MapKey() < found[j].MapKey() })
	return found
}

func (l *basicLoader) LoadEntry(c px.Context, name px.TypedName) px.LoaderEntry {
	return l.GetEntry(name)
}

func (l *basicLoader) GetEntry(name px.TypedName) px.LoaderEntry {
	l.lock.RLock()
	v := l.namedEntries[name.MapKey()]
	l.lock.RUnlock()
	return v
}

func (l *basicLoader) HasEntry(name px.TypedName) bool {
	l.lock.RLock()
	e, found := l.namedEntries[name.MapKey()]
	l.lock.RUnlock()
	return found && e.Value() != nil
}

func (l *basicLoader) SetEntry(name px.TypedName, entry px.LoaderEntry) px.LoaderEntry {
	l.lock.Lock()
	defer l.lock.Unlock()

	if old, ok := l.namedEntries[name.MapKey()]; ok {
		ov := old.Value()
		if ov == nil {
			*old.(*loaderEntry) = *entry.(*loaderEntry)
			return old
		}
		nv := entry.Value()
		if ov == nv {
			return old
		}
		if ea, ok := ov.(px.Equality); ok && ea.Equals(nv, nil) {
			return old
		}

		if lt, ok := old.Value().(px.Type); ok {
			ob := bytes.NewBufferString(``)
			lt.ToString(ob, px.PrettyExpanded, nil)
			nb := bytes.NewBufferString(``)
			nv.(px.Type).ToString(nb, px.PrettyExpanded, nil)
			panic(px.Error(px.AttemptToRedefineType, issue.H{`name`: name, `old`: ob.String(), `new`: nb.String()}))
		}
		panic(px.Error(px.AttemptToRedefine, issue.H{`name`: name}))
	}
	l.namedEntries[name.MapKey()] = entry
	return entry
}

func (l *basicLoader) NameAuthority() px.URI {
	return px.RuntimeNameAuthority
}

func (l *parentedLoader) Discover(c px.Context, predicate func(tn px.TypedName) bool) []px.TypedName {
	found := l.parent.Discover(c, predicate)
	added := false
	for k := range l.namedEntries {
		tn := px.TypedNameFromMapKey(k)
		if !l.parent.HasEntry(tn) {
			if predicate(tn) {
				found = append(found, tn)
				added = true
			}
		}
	}
	if added {
		sort.Slice(found, func(i, j int) bool { return found[i].MapKey() < found[j].MapKey() })
	}
	return found
}

func (l *parentedLoader) HasEntry(name px.TypedName) bool {
	return l.parent.HasEntry(name) || l.basicLoader.HasEntry(name)
}

func (l *parentedLoader) LoadEntry(c px.Context, name px.TypedName) px.LoaderEntry {
	entry := l.parent.LoadEntry(c, name)
	if entry == nil || entry.Value() == nil {
		entry = l.basicLoader.LoadEntry(c, name)
	}
	return entry
}

func (l *parentedLoader) NameAuthority() px.URI {
	return l.parent.NameAuthority()
}

func (l *parentedLoader) Parent() px.Loader {
	return l.parent
}

func (l *typeSetLoader) Discover(c px.Context, predicate func(tn px.TypedName) bool) []px.TypedName {
	found := make([]px.TypedName, 0)
	ts := l.typeSet.Types()
	ts.EachKey(func(v px.Value) {
		tn := v.(px.TypedName)
		if predicate(tn) {
			found = append(found, tn)
		}
	})

	pf := l.parentedLoader.Discover(c, func(tn px.TypedName) bool { return !ts.IncludesKey(tn) && predicate(tn) })
	if len(pf) > 0 {
		found = append(found, pf...)
		sort.Slice(found, func(i, j int) bool { return found[i].MapKey() < found[j].MapKey() })
	}
	return found
}

func (l *typeSetLoader) HasEntry(name px.TypedName) bool {
	if _, ok := l.typeSet.GetType(name); ok {
		return true
	}
	if l.parentedLoader.HasEntry(name) {
		return true
	}
	if child, ok := name.RelativeTo(l.typeSet.TypedName()); ok {
		return l.HasEntry(child)
	}
	return false
}

func (l *typeSetLoader) LoadEntry(c px.Context, name px.TypedName) px.LoaderEntry {
	if tp, ok := l.typeSet.GetType(name); ok {
		return &loaderEntry{tp, nil}
	}
	entry := l.parentedLoader.LoadEntry(c, name)
	if entry == nil {
		if child, ok := name.RelativeTo(l.typeSet.TypedName()); ok {
			return l.LoadEntry(c, child)
		}
		entry = &loaderEntry{nil, nil}
		l.parentedLoader.SetEntry(name, entry)
	}
	return entry
}

func (l *typeSetLoader) SetEntry(name px.TypedName, entry px.LoaderEntry) px.LoaderEntry {
	return l.parent.(px.DefiningLoader).SetEntry(name, entry)
}

func (l *typeSetLoader) TypeSet() px.Type {
	return l.typeSet
}
