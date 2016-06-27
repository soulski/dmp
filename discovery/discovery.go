package discovery

type Discovery interface {
	ReadLocalService() *Service
	ReadNS(namespace string) []*Service
	ReadAll() []*Service
	ReadMultiNS(namespaces []string) map[string][]*Service
	ReadSubscriber(topic string) map[string][]*Service

	Register(ns string, commPort uint16) error
	Unregister() error

	Update(ns string, commPort uint16) error

	SubscribeTopic(topicName string) error
	UnsubscribeTopic(topicName string) error

	Start() (chan bool, error)
	Stop() error
}
