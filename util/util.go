package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
)

func FindAvailableTCPPort(host string) (int, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:0", host))
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, err
}

func HTTPPut(url string, msg []byte) ([]byte, error) {
	req, err := http.NewRequest("PUT", url, bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	resBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return resBytes, nil
}
