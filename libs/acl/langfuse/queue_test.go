package langfuse

import (
	"math/rand"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	q := newQueue(500)
	for i := 0; i < 500; i++ {
		go func() {
			time.Sleep(time.Duration(rand.Int63n(10)) * time.Millisecond)
			success := q.put(&event{})
			if !success {
				t.Error("put failed")
				return
			}
		}()
	}
	for i := 0; i < 500; i++ {
		go func() {
			time.Sleep(time.Duration(rand.Int63n(10)) * time.Millisecond)
			q.get(time.Second)
			q.done()
		}()
	}
	q.join()
}
