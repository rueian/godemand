package plugin

import (
	"encoding/json"
	"net/rpc"
)

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
