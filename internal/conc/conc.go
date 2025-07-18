package conc

import "sync"

type Locked[T any] struct {
	inner T
	mu    *sync.Mutex
}

func NewLocked[T any](inner T) *Locked[T] {
	return &Locked[T]{
		inner: inner,
		mu:    &sync.Mutex{},
	}
}

func (l *Locked[T]) Get() T {
	l.mu.Lock()
	return l.inner
}

func (l *Locked[T]) Release() {
	l.mu.Unlock()
}
