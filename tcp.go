package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/soulski/dmp/comm"
)

func client(url string) {
	sender, err := comm.Dial(url)
	if err != nil {
		fmt.Println("Error : ", err)
		return
	}

	defer sender.Close()

	fmt.Println("Send : ", "Hi Server")
	err = sender.Send([]byte("Hi Server"))
	if err != nil {
		fmt.Println("Error : ", err)
		return
	}

	recv, err := sender.Recv()
	if err != nil {
		fmt.Println("Error : ", err)
		return
	}

	fmt.Println("Recv : ", string(recv))
}

func multiClient(urls []string) {
	sender, err := comm.MultiDial(urls)
	if err != nil {
		fmt.Println("Error : ", err)
		return
	}

	defer sender.Close()

	fmt.Println("Sent : Hi Server")
	err = sender.Send([]byte("Hi Server"))
	if err != nil {
		fmt.Println("Error : ", err)
		return
	}

	fmt.Println("Success")
}

type H struct {
}

func (h *H) Recv(msg []byte) ([]byte, error) {
	fmt.Println(string(msg))
	return []byte("Hi Client"), nil
}

func server(url string) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	addr, err := net.ResolveTCPAddr("tcp", url)
	bus, err := comm.CreateBus(addr, &H{}, logger)
	if err != nil {
		fmt.Println(err)
		return
	}

	bus.Start()
}

/*
func main() {
	if os.Args[1] == "server" {
		server(os.Args[2])
		os.Exit(0)
	}
	if os.Args[1] == "client" {
		client(os.Args[2])
		os.Exit(0)
	}
	if os.Args[1] == "multi" {
		multiClient(os.Args[2:])
		os.Exit(0)
	}
}
*/
