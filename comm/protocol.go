package comm

import (
	"github.com/soulski/dmp/util"
)

const (
	CONN_TYPE_INDEX = 0

	SYNC_FLAG  string = "0"
	ASYNC_FLAG string = "1"
)

type ConnType uint8

const (
	SYNC_CONN = iota
	ASYNC_CONN
)

type Protocol interface {
	AddEndpoint(*endpoint)
	RemoveEndpoint(*endpoint)

	Recv() (*Message, error)
	Send(*Message) error
}

/*

	Sender Protocol

*/

type Req struct {
	ep *endpoint
}

func CreateReq() *Req {
	return &Req{}
}

func (r *Req) AddEndpoint(ep *endpoint) {
	r.ep = ep
}

func (r *Req) Send(msg *Message) error {
	sync := []byte(SYNC_FLAG)
	msg.Header = append(msg.Header, sync...)
	return r.ep.Send(msg)
}

func (r *Req) Recv() (*Message, error) {
	return r.ep.Recv()
}

func (r *Req) RemoveEndpoint(ep *endpoint) {}

type Multi struct {
	eps []*endpoint
}

func CreateMulti() *Multi {
	return &Multi{
		eps: []*endpoint{},
	}
}

func (m *Multi) AddEndpoint(ep *endpoint) {
	m.eps = append(m.eps, ep)
}

func (m *Multi) RemoveEndpoint(ep *endpoint) {

}

func (m *Multi) Send(msg *Message) error {
	ackCh := make(chan string)
	ackNum := len(m.eps)

	async := []byte(ASYNC_FLAG)
	msg.Header = append(msg.Header, async...)

	defer close(ackCh)

	for _, ep := range m.eps {
		go func(ep *endpoint, ackCh chan string) {
			var err error
			var rMsg *Message

			sMsg := msg.Dup()
			defer sMsg.Free()

			if err = ep.Send(sMsg); err != nil {
				ackCh <- ep.RemoteAddr().String()
				return
			}

			if rMsg, err = ep.Recv(); err != nil {
				ackCh <- ep.RemoteAddr().String()
				return
			} else {
				rMsg.Free()
			}

			ackCh <- ""
		}(ep, ackCh)
	}

	allRecv := true
	failNodes := []string{}

	for index := 0; index < ackNum; index++ {
		ack := <-ackCh
		if ack != "" {
			failNodes = append(failNodes, ack)
			allRecv = false
		}
	}

	if !allRecv {
		return util.CreateIncompleteMultiErr(failNodes)
	}

	return nil
}

func (m *Multi) Recv() (*Message, error) {
	return nil, nil
}

type Noti struct {
	ep *endpoint
}

func CreateNoti() *Noti {
	return &Noti{}
}

func (r *Noti) AddEndpoint(ep *endpoint) {
	r.ep = ep
}

func (r *Noti) Send(msg *Message) error {
	sync := []byte(ASYNC_FLAG)
	msg.Header = append(msg.Header, sync...)
	return r.ep.Send(msg)
}

func (r *Noti) Recv() (*Message, error) {
	return r.ep.Recv()
}

func (r *Noti) RemoveEndpoint(ep *endpoint) {}

/*

	Receiver Protocol

*/

type Res struct {
	ep       *endpoint
	connType ConnType
}

func CreateRes() *Res {
	return &Res{}
}

func (r *Res) AddEndpoint(ep *endpoint) {
	r.ep = ep
}

func (r *Res) Recv() (*Message, error) {
	msg, err := r.ep.Recv()
	if err != nil {
		return nil, err
	}

	header := string(msg.Header)
	if header == ASYNC_FLAG {
		r.connType = ASYNC_CONN
	} else {
		r.connType = SYNC_CONN
	}

	switch r.connType {
	case ASYNC_CONN:
		return r.asynRecv(msg)
	case SYNC_CONN:
		r.connType = SYNC_CONN
		return r.syncRecv(msg)
	}

	return nil, util.CreateInvalidProtocol("Unknow connection type flag")
}

func (r *Res) asynRecv(recvMsg *Message) (*Message, error) {
	reply := []byte("ACKS")
	sMsg := CreateMessage(reply)
	defer sMsg.Free()

	return recvMsg, r.ep.Send(sMsg)
}

func (r *Res) syncRecv(recvMsg *Message) (*Message, error) {
	return recvMsg, nil
}
func (r *Res) Send(msg *Message) error {
	switch r.connType {
	case ASYNC_CONN:
		return nil
	case SYNC_CONN:
		return r.ep.Send(msg)
	}

	return util.CreateInvalidProtocol("Unknow connection type flag")
}

func (r *Res) RemoveEndpoint(ep *endpoint) {}
