package pximpl

import (
	"fmt"

	"github.com/lyraproj/pcore/px"
)

type (
	setting struct {
		name         string
		value        px.Value
		defaultValue px.Value
		valueType    px.Type
	}
)

func (s *setting) get() px.Value {
	return s.value
}

func (s *setting) reset() {
	s.value = s.defaultValue
}

func (s *setting) set(value px.Value) {
	if !px.IsInstance(s.valueType, value) {
		panic(px.DescribeMismatch(fmt.Sprintf(`Setting '%s'`, s.name), s.valueType, px.DetailedValueType(value)))
	}
	s.value = value
}

func (s *setting) isSet() bool {
	return s.value != nil // As opposed to UNDEF which is a proper value
}
