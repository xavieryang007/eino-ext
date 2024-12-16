package langfuse

import "time"

type options struct {
	threads          int
	timeout          time.Duration
	maxTaskQueueSize int
	flushAt          int
	flushInterval    time.Duration
	sampleRate       float64
	logMessage       string
	maskFunc         func(string) string
	maxRetry         uint64
}

type Option func(*options)

func WithThreads(threads int) Option {
	return func(o *options) {
		o.threads = threads
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

func WithMaxTaskQueueSize(maxTaskQueueSize int) Option {
	return func(o *options) {
		o.maxTaskQueueSize = maxTaskQueueSize
	}
}

func WithFlushInterval(flushInterval time.Duration) Option {
	return func(o *options) {
		o.flushInterval = flushInterval
	}
}

func WithSampleRate(sampleRate float64) Option {
	return func(o *options) {
		o.sampleRate = sampleRate
	}
}

func WithLogMessage(logMessage string) Option {
	return func(o *options) {
		o.logMessage = logMessage
	}
}

func WithMaskFunc(maskFunc func(string) string) Option {
	return func(o *options) {
		o.maskFunc = maskFunc
	}
}

func WithMaxRetry(maxRetry uint64) Option {
	return func(o *options) {
		o.maxRetry = maxRetry
	}
}
