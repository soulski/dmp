package comm

type Socket interface {
	Send([]byte) error
	Recv() ([]byte, error)
	Close() error
}
