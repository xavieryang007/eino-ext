package langfuse

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

func newTaskManager(
	threads int,
	cli *http.Client,
	host string,
	maxTaskQueueSize int,
	flushAt int,
	flushInterval time.Duration,
	sampleRate float64,
	logMessage string,
	maskFunc func(string) string,
	sdkName string,
	sdkVersion string,
	sdkIntegration string,
	publicKey string,
	secretKey string,
	maxRetry uint64,
) *taskManager {
	langfuseCli := newClient(cli, host, publicKey, secretKey, sdkVersion)
	q := newQueue(maxTaskQueueSize)
	if threads < 1 {
		threads = 1
	}
	wg := &sync.WaitGroup{}
	for i := 0; i < threads; i++ {
		newIngestionConsumer(langfuseCli, q, flushAt, flushInterval, sampleRate, logMessage, maskFunc, sdkName, sdkVersion, sdkIntegration, publicKey, maxRetry, wg).run()
	}

	return &taskManager{q: q, mediaWG: wg}
}

type taskManager struct {
	q       *queue
	mediaWG *sync.WaitGroup
}

func (t *taskManager) push(e *event) error {
	e.TimeStamp = time.Now()
	success := t.q.put(e)
	if !success {
		return errors.New("event send queue is full")
	}
	return nil
}

func (t *taskManager) flush() {
	t.q.join()
	t.mediaWG.Wait()
}
