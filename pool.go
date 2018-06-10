package gopool

import (
	"errors"
	"sync"
)

type signal struct {
	msg string
}

type f func() error

// Pool is the container accepts tasks
// with idle workers, Pool fetch one and make it deal with the task
// without idle workers and the Pool is not full, a new worker will be generated to handle the task
// without idle workers and the Poll is full, the task will hang
type Pool struct {
	capacity         int64
	running          int64
	availableWorkers []*Worker
	idleSingalChan   chan signal
	closeSignalChan  chan signal
	mu               sync.Mutex
	once             sync.Once
}

// Submit a task to pool
func (p *Pool) Submit(t f) error {
	if len(p.closeSignalChan) > 0 {
		return errors.New("pool is already closed")
	}
	w := p.getWorker()
	w.sendTask(t)
	return nil
}

func (p *Pool) getWorker() *Worker {
	stuck := false
	var w *Worker

	p.mu.Lock()
	workers := p.availableWorkers
	lastWorkerIndex := len(workers) - 1
	if lastWorkerIndex < 0 { // no available workers
		if p.running >= p.capacity {
			stuck = true
		} else {
			p.running++
		}
	} else {
		p.running++
		w = workers[lastWorkerIndex]
		workers[lastWorkerIndex] = nil
		p.availableWorkers = workers[:lastWorkerIndex]
	}
	p.mu.Unlock()

	if stuck {
		<-p.idleSingalChan
		for {
			p.mu.Lock()
			workers2 := p.availableWorkers
			lastWorkerIndex2 := len(workers2) - 1
			if lastWorkerIndex2 < 0 {
				p.mu.Unlock()
				continue
			}
			p.running++
			w = workers2[lastWorkerIndex2]
			workers2[lastWorkerIndex2] = nil
			p.availableWorkers = workers2[:lastWorkerIndex2]
			p.mu.Unlock()
		}
	} else if w == nil {
		w = &Worker{
			pool:  p,
			tasks: make(chan f),
		}
	}
	w.run()
	return w
}

func (p *Pool) putWorker(w *Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.availableWorkers = append(p.availableWorkers, w)
	p.idleSingalChan <- signal{}
}
