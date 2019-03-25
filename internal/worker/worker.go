package worker

import (
	"github.com/sirupsen/logrus"
)

type Worker struct {
	id         int
	jobQueue   chan Job
	workerPool chan chan Job
	quitChan   chan bool

	logger logrus.FieldLogger
}

func NewWorker(id int, workerPool chan chan Job, logger logrus.FieldLogger) Worker {
	return Worker{
		id:         id,
		jobQueue:   make(chan Job),
		workerPool: workerPool,
		quitChan:   make(chan bool),

		logger: logger.WithField("component", "worker"),
	}
}

func (w Worker) start() {
	go func() {
		for {
			w.workerPool <- w.jobQueue

			select {
			// Dispatcher added a job to jobQueue.
			case job := <-w.jobQueue:
				w.logger.Debugf("Starting job %s", job.Name())
				if err := job.Do(); err != nil {
					w.logger.Error("error while processing %s: %v", job.Name(), err)
				}
			case <-w.quitChan:
				w.logger.Infof("Stopping Worker %04d", w.id)
			}
		}
	}()
}

func (w Worker) stop() {
	go func() {
		w.quitChan <- true
	}()
}

type Dispatcher struct {
	workerPool chan chan Job
	maxWorkers int
	jobQueue   chan Job

	logger logrus.FieldLogger
}

func NewDispatcher(jobQueue chan Job, maxWorkers int, logger logrus.FieldLogger) *Dispatcher {
	workerPool := make(chan chan Job, maxWorkers)

	return &Dispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		workerPool: workerPool,

		logger: logger.WithField("component", "dispatcher"),
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(i+1, d.workerPool, d.logger)
		worker.start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			d.logger.Debugf("Job enqueued: %s", job.Name())
			go func() {
				workerJobQueue := <-d.workerPool
				workerJobQueue <- job
			}()
		}
	}
}
