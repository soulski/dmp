package discovery

import (
	"net"
)

type NetworkType string

const (
	LanNetwork   NetworkType = "lan"
	WanNetwork   NetworkType = "wan"
	LocalNetwork NetworkType = "local"
)

type Config struct {
	Name    string
	Addr    *net.TCPAddr
	Network NetworkType
}
