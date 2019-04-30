package plugin

import (
	"bufio"
	"context"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

const CurrentProtocolVersion = 1
const RPCServerName = "Controller"

var MinimumProtocolVersion = 1

var (
	ProtocolVersionTooOldErr = xerrors.New("plugin's protocol version is too old")
	LaunchTimeoutErr         = xerrors.New("plugin doesn't print its port in time")
	MalformedLaunchSignError = xerrors.New("plugin prints a malformed sign")
)

type CmdParam struct {
	Name string
	Path string
	Envs []string
}

func NewLauncher(param CmdParam, logger *log.Logger) *Launcher {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	return &Launcher{CmdParam: param, logger: logger}
}

type Launcher struct {
	CmdParam   CmdParam
	Controller Controller
	command    *exec.Cmd
	client     *rpc.Client
	cancel     context.CancelFunc
	logger     *log.Logger
	doneCh     chan error
	err        error
}

func (l *Launcher) Launch() (Controller, error) {
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	cmd := exec.CommandContext(ctx, l.CmdParam.Path)
	cmd.Env = append(os.Environ(), l.CmdParam.Envs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	l.command = cmd
	l.doneCh = make(chan error, 1)
	listenCh := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			l.logger.Println(l.CmdParam.Name + " stdout: " + scanner.Text())
			if strings.HasPrefix(scanner.Text(), ListenedSign) {
				listenCh <- scanner.Text()
				close(listenCh)
			}
		}
		if scanner.Err() != nil {
			l.logger.Println(l.CmdParam.Name + " stdout: " + scanner.Err().Error())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			l.logger.Println(l.CmdParam.Name + " stderr: " + scanner.Text())
		}
		if scanner.Err() != nil {
			l.logger.Println(l.CmdParam.Name + " stderr: " + scanner.Err().Error())
		}
	}()
	go func() {
		if err := cmd.Wait(); err != nil {
			l.doneCh <- err
			l.logger.Println(l.CmdParam.Name + ": " + err.Error())
		}
		close(l.doneCh)
	}()

	var network, address, version string
	select {
	case sign := <-listenCh:
		s := strings.Split(sign, "|")
		if len(s) != 4 {
			cancel()
			return nil, xerrors.Errorf("fail to parse sign %q: %w", sign, MalformedLaunchSignError)
		}
		version, network, address = s[1], s[2], s[3]
	case <-time.Tick(30 * time.Second):
		cancel()
		return nil, xerrors.Errorf("fail to connect plugin in 30 sec: %w", LaunchTimeoutErr)
	}

	if v, err := strconv.Atoi(version); err != nil || v < MinimumProtocolVersion {
		cancel()
		return nil, xerrors.Errorf("fail to load the plugin %s: %w", l.CmdParam.Name, ProtocolVersionTooOldErr)
	}

	l.client, err = rpc.Dial(network, address)
	if err != nil {
		cancel()
		return nil, err
	}

	l.Controller = &rpcClient{client: l.client}
	return l.Controller, nil
}

func (l *Launcher) Err() error {
	for {
		err, more := <-l.doneCh
		if err != nil {
			l.err = err
			return l.err
		}
		if !more {
			return l.err
		}
	}
}

func (l *Launcher) Close() {
	if l.client != nil {
		l.client.Close()
	}
	if l.cancel != nil {
		l.cancel()
	}
}

type rpcClient struct {
	client *rpc.Client
}

func (c *rpcClient) FindResource(pool types.ResourcePool, params map[string]interface{}) (res types.Resource, err error) {
	err = call(c.client, RPCServerName+".FindResource", &FindResourceArgs{Pool: pool, Params: params}, &res)
	return
}

func (c *rpcClient) SyncResource(resource types.Resource, params map[string]interface{}) (res types.Resource, err error) {
	err = call(c.client, RPCServerName+".SyncResource", &SyncResourceArgs{Resource: resource, Params: params}, &res)
	return
}
