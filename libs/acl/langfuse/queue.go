package langfuse

import (
	"sync"
	"time"
)

const (
	defaultMaxSize = 100
)

func newQueue(maxSize int) *queue {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return &queue{
		data:  make(chan *event, maxSize),
		empty: sync.NewCond(&sync.Mutex{}),
	}
}

type queue struct {
	data       chan *event
	empty      *sync.Cond
	unfinished int
}

func (q *queue) put(value *event) bool {
	q.empty.L.Lock()
	defer q.empty.L.Unlock()
	for {
		select {
		case q.data <- value:
			q.unfinished++
			return true
		default:
			return false
		}
	}
}

func (q *queue) get(timeout time.Duration) (*event, bool) {
	select {
	case v := <-q.data:
		return v, true
	case <-time.After(timeout):
		return nil, false
	}
}

func (q *queue) done() {
	q.empty.L.Lock()
	defer q.empty.L.Unlock()
	q.unfinished--
	if q.unfinished == 0 {
		q.empty.Broadcast()
	}
}

func (q *queue) join() {
	q.empty.L.Lock()
	defer q.empty.L.Unlock()
	for q.unfinished > 0 {
		q.empty.Wait()
	}
}
