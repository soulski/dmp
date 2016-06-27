package comm

import (
	"net"
)

type endpoint struct {
	pipe *pipe
	addr net.Addr
}

func createEndpoint(conn *net.TCPConn) *endpoint {
	raddr := conn.RemoteAddr()
	return &endpoint{
		pipe: createPipe(conn),
		addr: raddr,
	}
}

func (e *endpoint) Send(msg *Message) error {
	return e.pipe.Send(msg)
}

func (e *endpoint) Recv() (*Message, error) {
	return e.pipe.Recv()
}

func (e *endpoint) RemoteAddr() net.Addr {
	return e.addr
}

func (e *endpoint) Close() error {
	return e.pipe.Close()
}
