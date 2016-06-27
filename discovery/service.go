package discovery

import (
	"net"
)

type ServiceStatus int

const (
	ServiceAlive ServiceStatus = iota
	ServiceFail
	ServiceSuspect
)

var serviceStatusName = map[ServiceStatus]string{
	ServiceAlive:   "Alive",
	ServiceFail:    "Out of service",
	ServiceSuspect: "Suspect",
}

func (s ServiceStatus) String() string {
	return serviceStatusName[s]
}

type Service struct {
	Namespace string
	IP        net.IP
	CommPort  uint16
	Topic     map[string]bool
	Status    ServiceStatus
}

func CreateService(ns string, ip net.IP, commPort uint16, status ServiceStatus) *Service {
	return &Service{
		Namespace: ns,
		IP:        ip,
		CommPort:  commPort,
		Topic:     make(map[string]bool),
		Status:    status,
	}
}

func (s *Service) GetCommAddr() *net.TCPAddr {
	return &net.TCPAddr{
		IP:   s.IP,
		Port: int(s.CommPort),
	}
}

func (s *Service) Subscribe(topic string) {
	s.Topic[topic] = true
}

func (s *Service) Unsubscribe(topic string) {
	delete(s.Topic, topic)
}
