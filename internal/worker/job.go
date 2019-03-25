package worker

type Job interface {
	Name() string
	Do() error
}

type job struct {
	name string
	do   func() error
}

func NewJob(name string, do func() error) Job {
	return &job{
		name: name,
		do:   do,
	}
}

func (j job) Name() string {
	return j.name
}

func (j job) Do() error {
	return j.do()
}
