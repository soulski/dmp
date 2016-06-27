package dmp

import (
	"sync"

	"github.com/soulski/dmp/discovery"
)

type Balance struct {
	Seeker    map[string]int
	indexLock sync.Mutex
}

func CreateBalance() *Balance {
	return &Balance{
		Seeker: make(map[string]int),
	}
}

func (b *Balance) Dispatch(namespace string, services []*discovery.Service) *discovery.Service {
	b.indexLock.Lock()

	index, ok := b.Seeker[namespace]
	if !ok {
		index = 0
		b.Seeker[namespace] = index
	}

	if index >= len(services) {
		index = 0
	}

	b.Seeker[namespace] = index + 1

	b.indexLock.Unlock()

	return services[index]
}
