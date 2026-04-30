package scheduler

type Scheduler interface {
	AddTask(name string, dependencies ...string) error
	RemoveTask(name string)
	Order() ([]string, error)
}
