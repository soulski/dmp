package discovery

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/serf/serf"
)

const (
	NAMESPACE_TAG = "namespace"
	COMM_PORT_TAG = "messagePort"
	TOPIC_TAG     = "topic"
)

type SerfDiscovery struct {
	conf      *Config
	syncPoint *SyncPoint

	serf        *serf.Serf
	serfEventCh chan serf.Event

	shutdownCh chan bool

	logger *log.Logger

	cache map[string][]*serf.Member
}

func CreateSerfDiscovery(conf *Config, syncPoint *SyncPoint, writer io.Writer) *SerfDiscovery {
	discovery := &SerfDiscovery{
		conf:        conf,
		serfEventCh: make(chan serf.Event),
		syncPoint:   syncPoint,
		logger:      log.New(writer, "", log.LstdFlags),
		cache:       make(map[string][]*serf.Member),
	}

	return discovery
}

type SyncPoint struct {
	Addresses []string
	CIDR      string
}

func CreateSyncPoint(addrs []string, cidr string) *SyncPoint {
	return &SyncPoint{
		Addresses: addrs,
		CIDR:      cidr,
	}
}

func (s *SyncPoint) GetCIDRAddresses(port uint16) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(s.CIDR)
	if err != nil {
		return nil, err
	}

	inc := func(ip net.IP) {
		for index := len(ip) - 1; index >= 0; index-- {
			ip[index]++
			if ip[index] > 0 {
				break
			}
		}
	}

	result := []string{}
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		result = append(result, fmt.Sprintf("%s:%d", ip, port))
	}

	return result, nil
}

func (s *SerfDiscovery) Start() (chan bool, error) {
	done := make(chan bool)

	serfConf := s.conf.serfConfig()
	serfConf.EventCh = s.serfEventCh

	serf, err := serf.Create(serfConf)
	if err != nil {
		return nil, err
	}

	s.serf = serf

	go func() {
		s.AutoJoin()
		done <- true
	}()

	go s.handleEvent()

	return done, nil
}

func (s *SerfDiscovery) Stop() error {
	s.logger.Println("[DMP][Info] Get signal shutdown")

	s.serf.Leave()
	if err := s.serf.Shutdown(); err != nil {
		s.logger.Println("[DMP][Error] cann't shutdown serf. force exits")
		return err
	}

	return nil
}

func (s *SerfDiscovery) SetLogWriter(logWriter io.Writer) {
	s.logger = log.New(logWriter, "", log.LstdFlags)
}

func (s *SerfDiscovery) AutoJoin() {
	s.logger.Printf("[DMP][Info]Start looking for contact point... \n")

	nodeCount := s.contactPointJoin()
	if nodeCount > 0 {
		s.logger.Printf("[DMP][Info]Success Join with %d nodes \n", nodeCount)
		return
	}

	nodeCount = s.contactCIDRJoin()
	if nodeCount > 0 {
		s.logger.Printf("[DMP][Info]Success Join with %d nodes\n", nodeCount)
		return
	}

	s.logger.Println("[DMP][Info]No exists cluster found...")
	s.logger.Println("[DMP][Info]Running alone in cluster...")
}

func (s *SerfDiscovery) contactPointJoin() (nodeCount int) {
	s.logger.Printf("[DMP][Info]ContactPoints %s lookup \n", s.syncPoint.Addresses)

	contactPoints := s.syncPoint.Addresses
	if contactPoints != nil {
		nodeCount = s.Join(contactPoints)
	}

	return nodeCount
}

func (s *SerfDiscovery) contactCIDRJoin() (nodeCount int) {
	s.logger.Printf("[DMP][Info]ContactCIDR %s lookup \n", s.syncPoint.CIDR)

	CIDR := s.syncPoint.CIDR
	if CIDR != "" {
		port := uint16(s.conf.Addr.Port)
		contactPoints, err := s.syncPoint.GetCIDRAddresses(port)
		fmt.Printf("[DMP][Debug]CIDR : %s", contactPoints)
		if err != nil {
			s.logger.Fatalf("[DMP][Error]Invalid CIDR : %s\n", err)
			return nodeCount
		}

		nodeCount = s.Join(contactPoints)
	}

	return nodeCount
}

func (s *SerfDiscovery) Join(contacts []string) int {
	nodeCount, err := s.serf.Join(contacts, false)
	if err != nil || nodeCount == 0 {
		return 0
	}

	return nodeCount
}

func (s *SerfDiscovery) handleEvent() {
	for {
		select {
		case event := <-s.serfEventCh:
			if IsModifyMemberEvent(event) {
				s.updateCache(event.(serf.MemberEvent))
			}
		case <-s.serf.ShutdownCh():
			s.Unregister()
			return
		}
	}
}

func (s *SerfDiscovery) updateCache(event serf.MemberEvent) {
	if serf.EventMemberFailed == event.Type {
		s.logger.Println("=========== Failed =============")
		s.logger.Println(event.Members)
	} else if serf.EventMemberLeave == event.Type {
		s.logger.Println("===========  Leave =============")
		s.logger.Println(event.Members)
	}
}

func (s *SerfDiscovery) updateService(service *Service) error {
	newTags := map[string]string{
		NAMESPACE_TAG: service.Namespace,
		COMM_PORT_TAG: strconv.Itoa(int(service.CommPort)),
	}

	for topic, _ := range service.Topic {
		newTags["TAG:"+topic] = topic
	}

	return s.serf.SetTags(newTags)
}

func (s *SerfDiscovery) Register(ns string, commPort uint16) error {
	member := s.serf.LocalMember()

	service := &Service{
		Namespace: ns,
		IP:        member.Addr,
		CommPort:  commPort,
	}

	return s.updateService(service)
}

func (s *SerfDiscovery) Update(ns string, commPort uint16) error {
	service := s.ReadLocalService()
	service.Namespace = ns
	service.CommPort = commPort

	return s.updateService(service)
}

func (s *SerfDiscovery) Unregister() error {
	return s.serf.SetTags(make(map[string]string))
}

func (s *SerfDiscovery) ReadLocalService() *Service {
	member := s.serf.LocalMember()
	serv, err := ConvertMemberToService(&member)
	if err != nil {
		s.logger.Println("[DMP][Warning] Error : ", err.Error())
	}

	return serv
}

func (s *SerfDiscovery) ReadAll() []*Service {
	members := s.serf.Members()
	services := make([]*Service, 0, len(members))

	for _, member := range members {
		_, found := member.Tags[NAMESPACE_TAG]
		if found {
			serv, err := ConvertMemberToService(&member)
			if err != nil {
				continue
			}
			s.logger.Println(serv)

			services = append(services, serv)
		}
	}

	return services
}

func (s *SerfDiscovery) ReadNS(namespace string) []*Service {
	ns := s.readServiceAlive(func(member *serf.Member) bool {
		ns, found := member.Tags[NAMESPACE_TAG]
		return found && ns == namespace
	})

	return ns[namespace]
}

func (s *SerfDiscovery) ReadMultiNS(namespaces []string) map[string][]*Service {
	return s.readServiceAlive(func(member *serf.Member) bool {
		ns, found := member.Tags[NAMESPACE_TAG]
		if found {
			for _, expectNs := range namespaces {
				if ns == expectNs {
					return true
				}
			}
		}

		return false
	})
}

func (s *SerfDiscovery) ReadSubscriber(topic string) map[string][]*Service {
	return s.readServiceAlive(func(member *serf.Member) bool {
		fTag := "TAG:" + topic
		_, ok := member.Tags[fTag]
		return ok
	})
}

func (s *SerfDiscovery) readServiceAlive(f func(*serf.Member) bool) map[string][]*Service {
	members := s.serf.Members()
	ns := map[string][]*Service{}

	for _, member := range members {
		if serf.StatusAlive != member.Status {
			continue
		}

		if !f(&member) {
			continue
		}

		serv, err := ConvertMemberToService(&member)
		if err != nil {
			s.logger.Println("[DMP][Warning] Error : ", err.Error())
			continue
		}

		services, ok := ns[serv.Namespace]
		if !ok {
			services = []*Service{}
			ns[serv.Namespace] = services
		}

		services = append(services, serv)
		ns[serv.Namespace] = services
	}

	return ns
}

func (s *SerfDiscovery) SubscribeTopic(topic string) error {
	lService := s.ReadLocalService()
	lService.Subscribe(topic)
	return s.updateService(lService)
}

func (s *SerfDiscovery) UnsubscribeTopic(topic string) error {
	lService := s.ReadLocalService()
	lService.Unsubscribe(topic)
	return s.updateService(lService)
}

/*
	Extend Config function to support serf
*/

func (c *Config) serfConfig() *serf.Config {
	serfConf := serf.DefaultConfig()

	switch c.Network {
	case "lan":
		serfConf.MemberlistConfig = memberlist.DefaultLANConfig()
	case "wan":
		serfConf.MemberlistConfig = memberlist.DefaultWANConfig()
	case "local":
		serfConf.MemberlistConfig = memberlist.DefaultLocalConfig()
	}

	if c.Name != "" {
		serfConf.NodeName = c.Name
	}
	if c.Addr.IP != nil {
		bindIP := c.Addr.IP.String()
		serfConf.MemberlistConfig.BindAddr = bindIP
		serfConf.MemberlistConfig.AdvertiseAddr = bindIP
	}
	if c.Addr.Port != 0 {
		port := int(c.Addr.Port)
		serfConf.MemberlistConfig.BindPort = port
		serfConf.MemberlistConfig.AdvertisePort = port
	}

	return serfConf
}

func ConvertMemberToService(member *serf.Member) (*Service, error) {
	commPort, err := strconv.ParseUint(member.Tags[COMM_PORT_TAG], 10, 16)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot get CommPort from member name %s \n", member.Name))
	}

	var status ServiceStatus
	switch member.Status {
	case serf.StatusAlive:
		status = ServiceAlive
	default:
		status = ServiceFail
	}

	service := CreateService(
		member.Tags[NAMESPACE_TAG],
		member.Addr,
		uint16(commPort),
		status,
	)

	for key, _ := range member.Tags {
		found := strings.Index(key, "TAG:")
		if found != -1 {
			service.Subscribe(strings.Split(key, ":")[1])
		}
	}

	return service, nil
}

func ConvertMembersToServices(members []*serf.Member) []*Service {
	services := make([]*Service, 0, len(members))
	for _, m := range members {
		service, err := ConvertMemberToService(m)
		if err != nil {
			continue
		}
		services = append(services, service)
	}

	return services
}

func IsModifyMemberEvent(event serf.Event) bool {
	switch event.EventType() {
	case serf.EventMemberJoin,
		serf.EventMemberLeave,
		serf.EventMemberFailed,
		serf.EventMemberUpdate:
		return true
	default:
		return false
	}
}
