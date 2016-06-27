package comm

import (
	"encoding/binary"
	"io"
	"net"
	"runtime/debug"

	"github.com/soulski/dmp/util"
)

type pipe struct {
	conn *net.TCPConn
}

func createPipe(conn *net.TCPConn) *pipe {
	return &pipe{
		conn: conn,
	}
}

func (p *pipe) Send(msg *Message) error {
	msgSize := uint64(len(msg.Body))
	headSize := uint64(len(msg.Header))

	var err error
	if err = binary.Write(p.conn, binary.BigEndian, headSize); err != nil {
		return err
	}

	if err = binary.Write(p.conn, binary.BigEndian, msgSize); err != nil {
		return err
	}

	if _, err = p.conn.Write(msg.Header); err != nil {
		return err
	}

	if _, err = p.conn.Write(msg.Body); err != nil {
		return err
	}

	return nil
}

func (p *pipe) Recv() (*Message, error) {
	var err error
	var msgSize int64
	var headSize int64

	if err = binary.Read(p.conn, binary.BigEndian, &headSize); err != nil {
		debug.PrintStack()
		return nil, err
	}

	if err = binary.Read(p.conn, binary.BigEndian, &msgSize); err != nil {
		debug.PrintStack()
		return nil, err
	}

	msg := ReqMessage(int(msgSize))

	msg.Header = msg.Header[0:headSize]
	msg.Body = msg.Body[0:msgSize]

	if headSize > HEADER_SIZE {
		return nil, util.CreateMsgTooLongErr(0, headSize)
	}

	if headSize != 0 {
		if _, err = io.ReadFull(p.conn, msg.Header); err != nil {
			msg.Free()
			debug.PrintStack()
			return nil, err
		}
	}

	if msgSize < 0 {
		return nil, util.CreateMsgTooLongErr(0, msgSize)
	}

	if _, err = io.ReadFull(p.conn, msg.Body); err != nil {
		msg.Free()
		debug.PrintStack()
		return nil, err
	}

	return msg, nil
}

func (p *pipe) Close() error {
	return p.conn.Close()
}
