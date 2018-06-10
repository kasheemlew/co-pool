package gopool

// Worker is the executor who handles the tasks
type Worker struct {
	pool  *Pool
	tasks chan f
}

func (w *Worker) sendTask(t f) {
	w.tasks <- t
}

func (w *Worker) stop() {
	w.sendTask(nil)
}

func (w *Worker) run() {
	for f := range w.tasks {
		if f != nil {
			f()
			w.pool.putWorker(w)
		}
		return
	}
}
