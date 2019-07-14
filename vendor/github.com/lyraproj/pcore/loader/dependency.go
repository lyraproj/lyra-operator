package loader

import "github.com/lyraproj/pcore/px"

type dependencyLoader struct {
	basicLoader
	loaders []px.ModuleLoader
	index   map[string]px.ModuleLoader
}

func newDependencyLoader(loaders []px.ModuleLoader) px.Loader {
	index := make(map[string]px.ModuleLoader, len(loaders))
	for _, ml := range loaders {
		n := ml.ModuleName()
		if n != `` {
			index[n] = ml
		}
	}
	return &dependencyLoader{
		basicLoader: basicLoader{namedEntries: make(map[string]px.LoaderEntry, 32)},
		loaders:     loaders,
		index:       index}
}

func init() {
	px.NewDependencyLoader = newDependencyLoader
}

func (l *dependencyLoader) LoadEntry(c px.Context, name px.TypedName) px.LoaderEntry {
	entry := l.basicLoader.LoadEntry(c, name)
	if entry == nil {
		entry = l.find(c, name)
		if entry == nil {
			entry = &loaderEntry{nil, nil}
		}
		l.SetEntry(name, entry)
	}
	return entry
}

func (l *dependencyLoader) LoaderFor(moduleName string) px.ModuleLoader {
	return l.index[moduleName]
}

func (l *dependencyLoader) find(c px.Context, name px.TypedName) px.LoaderEntry {
	if len(l.index) > 0 && name.IsQualified() {
		// Explicit loader for given name takes precedence
		if ml, ok := l.index[name.Parts()[0]]; ok {
			return ml.LoadEntry(c, name)
		}
	}

	for _, ml := range l.loaders {
		e := ml.LoadEntry(c, name)
		if !(e == nil || e.Value() == nil) {
			return e
		}
	}

	// Recursion or parallel go routines might have set the entry now. Returning
	// nil here might therefore be a lie.
	return l.basicLoader.LoadEntry(c, name)
}
