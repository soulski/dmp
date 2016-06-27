package comm

import (
	"sync/atomic"

	"github.com/soulski/dmp/util"
)

var poolByte = util.CreateBytePool()

const (
	HEADER_SIZE = 4
)

type Message struct {
	Header   []byte
	Body     []byte
	refCount int32

	cache bool
}

func ReqMessage(sz int) *Message {
	return &Message{
		Header: make([]byte, 0, HEADER_SIZE),
		Body:   make([]byte, 0, sz),
		cache:  true,
	}
}

func CreateMessage(body []byte) *Message {
	msg := ReqMessage(len(body))
	msg.Body = append(msg.Body, body...)
	return msg
}

func (m *Message) Dup() *Message {
	atomic.AddInt32(&m.refCount, 1)
	return m
}

func (m *Message) Free() {
	if m.cache {
		if v := atomic.AddInt32(&m.refCount, -1); v > 0 {
			return
		}

		//poolByte.Return(m.Header)
		//poolByte.Return(m.Body)
	}
}
