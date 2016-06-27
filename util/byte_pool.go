package util

import (
	"container/list"
	"sync"
	"time"
)

type cache struct {
	create time.Time
	data   []byte
}

type BytePool struct {
	pool   map[int]*list.List
	access map[int]*time.Time

	mapLock  sync.Mutex
	poolLock sync.Mutex
}

func CreateBytePool() *BytePool {
	now := time.Now()
	pool := BytePool{
		pool: map[int]*list.List{
			32:     new(list.List),
			128:    new(list.List),
			1024:   new(list.List),
			16384:  new(list.List),
			131072: new(list.List),
		},
		access: map[int]*time.Time{
			32:     &now,
			128:    &now,
			1024:   &now,
			16384:  &now,
			131072: &now,
		},
	}

	go pool.loopSweapOldCache()

	return &pool
}

func (p *BytePool) loopSweapOldCache() {
	for {
		time.Sleep(time.Minute)

		for sz, pool := range p.pool {
			p.mapLock.Lock()
			t := *p.access[sz]
			p.mapLock.Unlock()

			if time.Since(t) > time.Minute {
				p.clearPool(sz, pool.Len()/2)
			}
		}
	}
}

func (p *BytePool) clearPool(pIndex, size int) {
	pool := p.pool[pIndex]
	for index := 0; index < size; index++ {
		pool.Remove(pool.Back())
	}
}

func (p *BytePool) Get(size int) []byte {
	var fitSize int
	for sz, _ := range p.pool {
		if size <= sz {
			fitSize = sz
			break
		}
	}

	now := time.Now()

	p.mapLock.Lock()
	p.access[fitSize] = &now
	p.mapLock.Unlock()

	fitPool := p.pool[fitSize]

	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if fitPool == nil {
		return make([]byte, 0, size)
	} else if fitPool.Len() == 0 {
		cache := make([]byte, 0, size)
		fitPool.PushFront(cache)
		return cache
	}

	cache := fitPool.Front()
	b := cache.Value.([]byte)
	fitPool.Remove(cache)

	return b
}

func (p *BytePool) Return(cache []byte) {
	pool := p.pool[len(cache)]

	if pool != nil {
		pool.PushFront(cache)
	}

}
