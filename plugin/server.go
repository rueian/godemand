package plugin

import (
	"context"
	"encoding/json"
	"net"
	"net/rpc"
	"os"
	"strconv"

	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

var (
	TCPPortNotIntegerErr = xerrors.New(TCPPortEnvName + " should be integer")
)

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

	port := os.Getenv(TCPPortEnvName)
	if _, err := strconv.Atoi(port); err != nil {
		return xerrors.Errorf("fail to parse the port number: %q: %w", port, TCPPortNotIntegerErr)
	}

	l, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	s.Accept(l)
	return nil
}
