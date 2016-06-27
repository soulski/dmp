package dmp

import (
	"fmt"
	"io"
	"log"
	"net"
	"runtime/debug"

	"github.com/soulski/dmp/api"
	"github.com/soulski/dmp/api/res"
	"github.com/soulski/dmp/comm"
	"github.com/soulski/dmp/discovery"
	"github.com/soulski/dmp/util"
)

const (
	DEFAULT_COMM_PORT = 30000
)

type DMP struct {
	conf         *Config
	service      *discovery.Service
	contactPoint string

	api       *api.ApiServer
	discovery discovery.Discovery
	comm      *comm.Bus
	balance   *Balance

	logger *log.Logger
}

func CreateDMP(conf *Config, logWriter io.Writer) (*DMP, error) {
	logger := log.New(logWriter, "", log.LstdFlags)

	dmp := &DMP{}

	discConf, _ := conf.DiscoveryConfig()
	syncPoint := discovery.CreateSyncPoint(conf.ContactPoints, conf.ContactCIDR)
	discovery := discovery.CreateSerfDiscovery(
		discConf,
		syncPoint,
		logWriter,
	)

	commAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", conf.BindAddr, DEFAULT_COMM_PORT))
	if err != nil {
		return nil, err
	}

	apiServ := api.CreateApiServer(dmp, logger)

	comm, err := comm.CreateBus(commAddr, dmp, logger)
	if err != nil {
		return nil, err
	}

	dmp.discovery = discovery
	dmp.comm = comm
	dmp.api = apiServ
	dmp.conf = conf
	dmp.logger = logger
	dmp.balance = CreateBalance()

	return dmp, nil
}

func (d *DMP) Start() error {
	logger := d.logger

	dcDone, err := d.discovery.Start()
	if err != nil {
		logger.Printf("[DMP][ERROR] Error while start Discovery \n %s \n", err)
		return err
	}

	go d.comm.Start()
	logger.Printf("[DMP][Info]Start Communication running...")

	go d.api.Start()
	logger.Println("[DMP][Info]Public API running...")

	if dcDone != nil {
		<-dcDone
		logger.Printf("[DMP][Info]Start Discover running...")
	}

	logger.Printf("[DMP][Info]DMP is running")

	return nil
}

func (d *DMP) Stop() error {
	dcErr := d.discovery.Stop()
	if dcErr != nil {
		d.logger.Fatalf("[DMP][Warning] Error while stop discovery, force close discovery...")
	}

	apiErr := d.api.Stop()
	if apiErr != nil {
		d.logger.Fatalf("[DMP][Warning] Error while stop api, force close discovery...")
	}

	if dcErr != nil {
		return dcErr
	} else if apiErr != nil {
		return apiErr
	}

	d.comm.Stop()

	return nil
}

func (d *DMP) ListMembers(ns string) *res.Members {
	services := d.discovery.ReadNS(ns)

	members := make([]*res.Member, len(services))
	for index, service := range services {
		members[index] = &res.Member{
			IP:        service.IP.String(),
			Namespace: service.Namespace,
			Status:    service.Status.String(),
		}
	}

	return &res.Members{
		Members: members,
	}
}

func (d *DMP) ListAllMembers() *res.Members {
	services := d.discovery.ReadAll()

	members := make([]*res.Member, len(services))
	for index, service := range services {
		members[index] = &res.Member{
			IP:        service.IP.String(),
			Namespace: service.Namespace,
			Status:    service.Status.String(),
		}
	}

	return &res.Members{
		Members: members,
	}
}

func (d *DMP) ServiceRegister(ns string, contactPoint string) (*res.Member, error) {
	commAddr := d.comm.BusAddr()
	commPort := commAddr.Port

	if err := d.discovery.Register(ns, uint16(commPort)); err != nil {
		d.logger.Printf("[DMP][Warning] Error occur while register service : \n%s\n", err.Error())
		return nil, err
	}

	d.contactPoint = contactPoint

	ls := d.discovery.ReadLocalService()

	return &res.Member{
		IP:        ls.IP.String(),
		Namespace: ls.Namespace,
		Status:    ls.Status.String(),
	}, nil
}

func (d *DMP) ServiceUnregister() bool {
	if err := d.discovery.Unregister(); err != nil {
		d.logger.Printf("[DMP][Warning] Error occur while unregister service : \n%s\n", err.Error())
		return false
	}

	return true
}

func (d *DMP) SubscribeTopic(topicName string) bool {
	if err := d.discovery.SubscribeTopic(topicName); err != nil {
		d.logger.Printf("[DMP][Warning] Error subscribe topic : \n%s\n", err.Error())
		return false
	}
	return true
}

func (d *DMP) UnsubscribeTopic(topicName string) bool {
	if err := d.discovery.UnsubscribeTopic(topicName); err != nil {
		d.logger.Printf("[DMP][Warning] Error subscribe topic : \n%s\n", err.Error())
		return false
	}
	return true
}

func (d *DMP) Request(ns string, msg []byte) ([]byte, error) {
	services := d.discovery.ReadNS(ns)
	if len(services) <= 0 {
		return nil, fmt.Errorf("Error : namespace %s is not found.", ns)
	}

	service := d.balance.Dispatch(ns, services)

	sender, err := comm.DialWithAddr(service.GetCommAddr())
	defer sender.Close()

	if err != nil {
		debug.PrintStack()
		return nil, err
	}

	if err := sender.Send(msg); err != nil {
		debug.PrintStack()
		return nil, err
	}

	res, err := sender.Recv()
	if err != nil {
		debug.PrintStack()
		return nil, err
	}

	return res, nil
}

func (d *DMP) Publish(topic string, msg []byte) ([]byte, error) {
	addrs := []*net.TCPAddr{}
	nss := d.discovery.ReadSubscriber(topic)

	for ns, services := range nss {
		service := d.balance.Dispatch(ns, services)
		addrs = append(addrs, service.GetCommAddr())
	}

	if len(addrs) <= 0 {
		return nil, fmt.Errorf("Error : topic %s have no subscribe.", topic)
	}

	sender, err := comm.MultiDialAddr(addrs)
	defer sender.Close()

	if err != nil {
		return nil, err
	}

	if sender.Send(msg) != nil {
		return nil, err
	}

	return []byte("send"), nil
}

func (d *DMP) Notificate(ns string, msg []byte) ([]byte, error) {
	services := d.discovery.ReadNS(ns)
	if len(services) <= 0 {
		return nil, fmt.Errorf("Error : namespace %s is not found.", ns)
	}

	service := d.balance.Dispatch(ns, services)

	sender, err := comm.DialWithType(service.GetCommAddr(), comm.ASYNC)
	if err != nil {
		return nil, err
	}

	defer sender.Close()

	if sender.Send(msg) != nil {
		return nil, err
	}

	return sender.Recv()
}

func (d *DMP) Recv(msg []byte) ([]byte, error) {
	serviceRes, err := util.HTTPPut(d.contactPoint, msg)
	if err != nil {
		d.logger.Println("[DMP][Error] Error while connect with service")
		d.logger.Println("[DMP][Error] Error : ", err.Error())
		return nil, err
	}

	return serviceRes, err
}
