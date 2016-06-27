package dmp

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/soulski/dmp/discovery"
)

func DefaultConfig() *Config {
	return &Config{
		BindAddr:    "0.0.0.0",
		BindPort:    7946,
		NetworkType: "lan",
	}
}

type Config struct {
	NodeName      string
	BindAddr      string
	BindPort      int
	NetworkType   string
	ContactPoints []string
	ContactCIDR   string
	Namespace     string
	NetInterface  string
}

func (c *Config) Merge(optionConf *Config) {
	if c.NodeName == "" {
		c.NodeName = optionConf.NodeName
	}
	if c.BindAddr == "" {
		c.BindAddr = optionConf.BindAddr
	}
	if c.BindPort == 0 {
		c.BindPort = optionConf.BindPort
	}
	if c.ContactPoints == nil && optionConf.ContactPoints != nil {
		c.ContactPoints = make([]string, len(optionConf.ContactPoints))
		copy(c.ContactPoints, optionConf.ContactPoints)
	}
	if c.ContactCIDR == "" {
		c.ContactCIDR = optionConf.ContactCIDR
	}
	if c.Namespace == "" {
		c.Namespace = optionConf.Namespace
	}
	if c.NetInterface == "" {
		c.NetInterface = optionConf.NetInterface
	}
}

func (c *Config) DiscoveryConfig() (*discovery.Config, error) {
	addr, err := net.ResolveTCPAddr("tcp", c.BindAddr+":"+strconv.Itoa(c.BindPort))
	if err != nil {
		return nil, err
	}

	network := discovery.LocalNetwork
	switch c.NetworkType {
	case "lan":
		network = discovery.LanNetwork
	case "wan":
		network = discovery.WanNetwork
	case "local":
		network = discovery.LocalNetwork
	}

	return &discovery.Config{
		Name:    c.NodeName,
		Addr:    addr,
		Network: network,
	}, nil
}

func (c *Config) ParserContactCIDR() ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(c.ContactCIDR)
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
		result = append(result, fmt.Sprintf("%s:%d", ip, c.BindPort))
	}

	return result, nil
}

func (c *Config) GetBindAddr() (string, error) {
	if c.NetInterface != "" {
		iface, err := net.InterfaceByName(c.NetInterface)
		if err != nil {
			return "", err
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		if len(addrs) == 0 {
			return "", err
		}

		// If there is no bind IP, pick an address
		if c.BindAddr == "0.0.0.0" {
			found := false
			for _, a := range addrs {
				var addrIP net.IP
				addr, ok := a.(*net.IPNet)
				if !ok {
					continue
				}
				addrIP = addr.IP

				// Skip self-assigned IPs
				if addrIP.IsLinkLocalUnicast() {
					continue
				}

				// Found an IP
				return addrIP.String(), nil
			}
			if !found {
				return "", errors.New(fmt.Sprintf("Failed to find usable address for interface '%s'", c.NetInterface))
			}

		} else {
			// If there is a bind IP, ensure it is available
			found := false
			for _, a := range addrs {
				addr, ok := a.(*net.IPNet)
				if !ok {
					continue
				}
				if addr.IP.String() == c.BindAddr {
					found = true
					break
				}
			}
			if !found {
				return "", errors.New(fmt.Sprintf("Interface '%s' has no '%s' address",
					c.NetInterface, c.BindAddr))
			}
		}
	}

	return c.BindAddr, nil
}
