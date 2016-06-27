package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/soulski/dmp/dmp"
)

func main() {
	mainApp := cli.NewApp()
	mainApp.Name = "DMP"
	mainApp.Usage = "run decentralized message bus"
	mainApp.Version = "0.1.0"
	mainApp.Action = action
	mainApp.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "name, n",
			Value: "",
			Usage: "Node's name",
		},
		cli.StringFlag{
			Name:  "bind-host, host",
			Value: "0.0.0.0",
			Usage: "Address for bind service discovery",
		},
		cli.IntFlag{
			Name:  "bind-port, port",
			Value: 7946,
			Usage: "Port for bind service discovery",
		},
		cli.StringFlag{
			Name:  "network, net",
			Usage: "Type of network for adjust config to suite type (default lan)",
		},
		cli.StringSliceFlag{
			Name:  "contacts, c",
			Usage: "Specific Address of exists node in cluster to join cluster",
		},
		cli.StringFlag{
			Name:  "contact-cidr, cidr",
			Usage: "CIDR to join cluster",
		},
		cli.StringFlag{
			Name:  "namespace, ns",
			Value: "default",
			Usage: "Namespace",
		},
		cli.StringFlag{
			Name:  "net-if",
			Usage: "Network interface",
		},
	}

	mainApp.Run(os.Args)
}

func readConfig(c *cli.Context) *dmp.Config {
	conf := &dmp.Config{
		NodeName:      c.String("name"),
		BindAddr:      c.String("bind-host"),
		BindPort:      c.Int("bind-port"),
		NetworkType:   c.String("network"),
		ContactPoints: c.StringSlice("contacts"),
		ContactCIDR:   c.String("contact-cidr"),
		Namespace:     c.String("namespace"),
		NetInterface:  c.String("net-if"),
	}

	conf.Merge(dmp.DefaultConfig())

	return conf
}

func action(c *cli.Context) {
	conf := readConfig(c)

	if conf.BindAddr == "0.0.0.0" && conf.NetInterface != "" {
		bindAddr, err := conf.GetBindAddr()
		if err != nil {
			fmt.Printf("Error occur : %s", err)
			return
		}

		conf.BindAddr = bindAddr
	}

	dmp, err := dmp.CreateDMP(conf, os.Stdout)
	if err != nil {
		fmt.Printf("Error occur : %s", err)
		return
	}

	err = dmp.Start()
	if err == nil {
		shutdownCh := make(chan os.Signal, 1)
		signal.Notify(shutdownCh, os.Interrupt)

		for sig := range shutdownCh {
			fmt.Println(sig)

			if sig == syscall.SIGINT {
				if err := dmp.Stop(); err != nil {
					fmt.Printf("Error occur : %s", err)
				}

				os.Exit(0)
			}
		}
	} else {
		fmt.Printf("Error occur : %s", err)
	}

}
