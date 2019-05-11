package wf

import (
	"github.com/lyraproj/pcore/px"
)

type Reference interface {
	Step

	// Reference is the name of the activity that is referenced
	Reference() string
}

type reference struct {
	step
	referencedStep string
}

func MakeReference(name string, when Condition, input, output []px.Parameter, referencedStep string) Reference {
	return &reference{step{name, when, input, output}, referencedStep}
}

func (s *reference) Label() string {
	return `reference ` + s.name
}

func (s *reference) Reference() string {
	return s.referencedStep
}
