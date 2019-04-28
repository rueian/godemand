package plugin

import (
	"encoding/json"
	"net"
	"net/rpc"
	"strconv"
)

func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func call(client *rpc.Client, method string, arg, reply interface{}) (err error) {
	var bi, bo []byte
	if bi, err = json.Marshal(arg); err != nil {
		return
	}
	if err = client.Call(method, &bi, &bo); err != nil {
		return
	}
	err = json.Unmarshal(bo, reply)
	return
}
