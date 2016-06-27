package comm

import (
	"encoding/json"
	"log"
	"net"
)

type Receiver struct {
	proto Protocol
	eps   []*endpoint

	logger *log.Logger
}

func CreateReceiver(conn *net.TCPConn, logger *log.Logger) *Receiver {
	ep := createEndpoint(conn)

	res := CreateRes()
	res.AddEndpoint(ep)

	return &Receiver{
		proto:  res,
		logger: logger,
		eps:    []*endpoint{ep},
	}
}

func (r *Receiver) Recv() ([]byte, error) {
	msg, err := r.proto.Recv()
	if err != nil {
		return nil, err
	}

	rMsg := make([]byte, 0, len(msg.Body))
	rMsg = append(rMsg, msg.Body...)

	msg.Free()

	return rMsg, err
}

func (r *Receiver) Send(content []byte) error {
	msg := CreateMessage(content)
	defer msg.Free()

	return r.proto.Send(msg)
}

func (r *Receiver) SendJSON(obj interface{}) error {
	var raw []byte
	var err error

	if raw, err = json.Marshal(obj); err != nil {
		return err
	}

	return r.Send(raw)
}

func (r *Receiver) RecvJSON(t interface{}) error {
	raw, err := r.Recv()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, t); err != nil {
		return err
	}

	return nil
}

func (s *Receiver) Close() error {
	var err error

	for _, ep := range s.eps {
		err = ep.Close()
	}

	return err
}
