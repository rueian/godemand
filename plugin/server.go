package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"

	"github.com/rueian/godemand/types"
)

const ListenedSign = "PLUGIN_LISTENED"

type FindResourceArgs struct {
	Pool   types.ResourcePool
	Params map[string]interface{}
}

type SyncResourceArgs struct {
	Resource types.Resource
	Params   map[string]interface{}
}

type Server struct {
	controller Controller
}

func (*Server) ProtocolVersion(args *int, reply *int) error {
	*reply = CurrentProtocolVersion
	return nil
}

func (s *Server) FindResource(args *[]byte, reply *[]byte) error {
	var a FindResourceArgs
	if err := json.Unmarshal(*args, &a); err != nil {
		return err
	}
	res, err := s.controller.FindResource(a.Pool, a.Params)
	if err == nil {
		*reply, err = json.Marshal(res)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SyncResource(args *[]byte, reply *[]byte) error {
	var a SyncResourceArgs
	if err := json.Unmarshal(*args, &a); err != nil {
		return err
	}
	res, err := s.controller.SyncResource(a.Resource, a.Params)
	if err == nil {
		*reply, err = json.Marshal(res)
	}
	if err != nil {
		return err
	}
	return nil
}

func Serve(ctx context.Context, controller Controller) error {
	server := &Server{controller: controller}

	s := rpc.NewServer()

	if err := s.RegisterName(RPCServerName, server); err != nil {
		return err
	}

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}

	fmt.Printf("%s|%d|%s|%s\n", ListenedSign, CurrentProtocolVersion, l.Addr().Network(), l.Addr().String())
	os.Stdout.Sync()

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	s.Accept(l)
	return nil
}
