package api

type Workflow interface {
	Step

	Steps() []Step
}
