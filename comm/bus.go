package comm

import (
	"container/list"
	"log"
	"net"
	"sync"
)

type Handler interface {
	Recv([]byte) ([]byte, error)
}

type Bus struct {
	connPool *list.List
	listener *net.TCPListener
	handler  Handler

	logger *log.Logger
	close  bool

	poolLock sync.Mutex
}

func Listen(addr *net.TCPAddr) (*net.TCPListener, error) {
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		addr.Port = 0
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return nil, err
		}

		return l, nil
	}

	return l, nil
}

func CreateBus(addr *net.TCPAddr, handler Handler, logger *log.Logger) (*Bus, error) {
	ln, err := Listen(addr)
	if err != nil {
		return nil, err
	}

	bus := &Bus{
		connPool: list.New(),
		handler:  handler,
		listener: ln,
		close:    false,
		logger:   logger,
	}

	return bus, err
}

func (b *Bus) Start() {
	for {
		if b.close {
			break
		}

		conn, err := b.listener.AcceptTCP()
		if err != nil {
			b.logger.Println("[DMP][Error] ", err)
			break
		}

		go func(conn *net.TCPConn) {
			b.poolLock.Lock()
			ele := b.connPool.PushFront(conn)
			b.poolLock.Unlock()

			HandleReceive(conn, b.handler, b.logger)

			b.poolLock.Lock()
			b.connPool.Remove(ele)
			b.poolLock.Unlock()
		}(conn)
	}
}

func (b *Bus) Stop() {
	b.close = true
	b.listener.Close()

	for e := b.connPool.Front(); e != nil; e = e.Next() {
		conn := e.Value.(*net.TCPConn)
		conn.Close()
	}
}

func (b *Bus) BusAddr() *net.TCPAddr {
	return b.listener.Addr().(*net.TCPAddr)
}

func HandleReceive(conn *net.TCPConn, handler Handler, logger *log.Logger) {
	recv := CreateReceiver(conn, logger)
	defer recv.Close()

	msg, err := recv.Recv()
	if err != nil {
		logger.Println("[DMP][Error]", err)
		return
	}

	res, err := handler.Recv(msg)
	if err != nil {
		logger.Println("[DMP][Error]", err)
		return
	}

	err = recv.Send(res)
	if err != nil {
		logger.Println("[DMP][Error] ", err)
		return
	}
}
