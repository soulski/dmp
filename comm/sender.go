package comm

import (
	"encoding/json"
	"net"
)

type ReqType int

const (
	SYNC ReqType = iota
	ASYNC
)

type Sender struct {
	proto Protocol
	eps   []*endpoint
}

func Dial(url string) (*Sender, error) {
	raddr, err := net.ResolveTCPAddr("tcp", url)
	if err != nil {
		return nil, err
	}

	return DialWithAddr(raddr)
}

func DialWithAddr(addr *net.TCPAddr) (*Sender, error) {
	return DialWithType(addr, SYNC)
}

func DialWithType(addr *net.TCPAddr, rType ReqType) (*Sender, error) {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	var proto Protocol

	switch rType {
	case SYNC:
		proto = CreateReq()
	case ASYNC:
		proto = CreateNoti()
	}

	ep := createEndpoint(conn)
	proto.AddEndpoint(ep)

	return &Sender{
		proto: proto,
		eps:   []*endpoint{ep},
	}, nil
}

func MultiDial(urls []string) (*Sender, error) {
	addrs := make([]*net.TCPAddr, len(urls))
	for _, url := range urls {
		addr, err := net.ResolveTCPAddr("tcp", url)
		if err != nil {
			return nil, err
		}

		addrs = append(addrs, addr)
	}

	return MultiDialAddr(addrs)
}

func MultiDialAddr(addrs []*net.TCPAddr) (*Sender, error) {
	multi := CreateMulti()
	eps := make([]*endpoint, 0, len(addrs))

	for _, addr := range addrs {
		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			return nil, err
		}

		ep := createEndpoint(conn)
		eps = append(eps, ep)

		multi.AddEndpoint(ep)
	}

	return &Sender{
		proto: multi,
		eps:   eps,
	}, nil
}

func (s *Sender) Send(content []byte) error {
	msg := CreateMessage(content)
	defer msg.Free()

	return s.proto.Send(msg)
}

func (s *Sender) Recv() ([]byte, error) {
	msg, err := s.proto.Recv()
	if err != nil {
		return nil, err
	}

	defer msg.Free()

	rMsg := make([]byte, 0, len(msg.Body))
	rMsg = append(rMsg, msg.Body...)

	return rMsg, nil
}

func (s *Sender) SendJSON(obj interface{}) error {
	var raw []byte
	var err error

	if raw, err = json.Marshal(obj); err != nil {
		return err
	}

	return s.Send(raw)
}

func (s *Sender) RecvJSON(t interface{}) error {
	raw, err := s.Recv()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, t); err != nil {
		return err
	}

	return nil
}

func (s *Sender) Close() error {
	var err error

	for _, ep := range s.eps {
		err = ep.Close()
	}

	return err
}
