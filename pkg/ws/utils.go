package ws

import (
	"maps"
	"sync"
)

type SafeInt struct {
	mu sync.RWMutex
	v  int
}

func NewSafeInt(initial int) SafeInt {
	return SafeInt{
		v: initial,
	}
}

func (p *SafeInt) Get() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v
}

func (p *SafeInt) Set(value int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.v = value
}

func (p *SafeInt) Inc() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.v++
}

func (p *SafeInt) Dec() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.v--
}

type SafeMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewSafeMap[K comparable, V any]() SafeMap[K, V] {
	return SafeMap[K, V]{
		m: make(map[K]V),
	}
}

func (p *SafeMap[K, V]) Get(key K) (V, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.m[key]
	return v, ok
}

func (p *SafeMap[K, V]) GetAll() map[K]V {
	p.mu.RLock()
	defer p.mu.RUnlock()
	copy := make(map[K]V)
	maps.Copy(copy, p.m)
	return copy
}

func (p *SafeMap[K, V]) Set(key K, value V) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m[key] = value
}

func (p *SafeMap[K, V]) Delete(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.m, key)
}

func (p *SafeMap[K, V]) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.m)
}

func (p *SafeMap[K, V]) GetOrCreate(key K, factory func() V) V {
	p.mu.Lock()
	defer p.mu.Unlock()

	if v, exists := p.m[key]; exists {
		return v
	}

	v := factory()
	p.m[key] = v
	return v
}

func (p *SafeMap[K, V]) ForEach(fn func(K, *V)) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for k, v := range p.m {
		fn(k, &v)
	}
}
