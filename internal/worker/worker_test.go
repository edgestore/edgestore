package worker

import (
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = ioutil.Discard
	return logger
}

func TestNewWorker(t *testing.T) {
	workerPool := make(chan chan Job)
	nw := NewWorker(1, workerPool, newLogger())
	assert.Equal(t, nw.id, 1)
	assert.NotNil(t, nw.jobQueue)
	assert.NotNil(t, nw.workerPool)
	assert.NotNil(t, nw.quitChan)
}

func TestNewWorker_stop(t *testing.T) {
	workerPool := make(chan chan Job)
	nw := NewWorker(1, workerPool, newLogger())
	nw.stop()
	res := <-nw.quitChan
	assert.Equal(t, res, true)
}
func TestNewDispatcher(t *testing.T) {
	jobQueue := make(chan Job)
	ds := NewDispatcher(jobQueue, 2, newLogger())
	if assert.NotNil(t, ds) {
		assert.Equal(t, ds.maxWorkers, 2)
		assert.NotNil(t, ds.workerPool)
		assert.NotNil(t, ds.jobQueue)
	}
}
