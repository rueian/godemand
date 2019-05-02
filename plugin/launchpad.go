package plugin

import (
	"strings"
	"sync"

	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

var ControllerNotFoundErr = xerrors.New("Controller not found in launchpad")

type Errors struct {
	errs []error
}

func (e *Errors) Error() string {
	m := make([]string, len(e.errs))
	for i, err := range e.errs {
		m[i] = err.Error()
	}
	return strings.Join(m, " ")
}

func (e *Errors) Append(err error) {
	e.errs = append(e.errs, err)
}

func (e *Errors) Len() int {
	return len(e.errs)
}

func NewLaunchpad() *Launchpad {
	return &Launchpad{
		launchers: make(map[string]*Launcher),
	}
}

type Launchpad struct {
	launchers map[string]*Launcher
	mu        sync.Mutex
}

func (p *Launchpad) SetLaunchers(params map[string]types.CmdParam) error {
	p.mu.Lock()
	for k, l := range p.launchers {
		if param, ok := params[k]; !ok || changed(l.CmdParam, param) {
			l.Close()
			delete(p.launchers, k)
		}
	}
	p.mu.Unlock()

	var errs Errors
	for k, param := range params {
		p.mu.Lock()
		if _, ok := p.launchers[k]; !ok {
			p.mu.Unlock()

			launcher := NewLauncher(param, nil)
			_, err := launcher.Launch()
			if err != nil {
				errs.Append(err)
			} else {
				p.mu.Lock()
				p.launchers[k] = launcher
				go func() {
					if err := launcher.Err(); err != nil {
						// TODO: logging
					}
					p.mu.Lock()
					defer p.mu.Unlock()
					launcher.Close()
					delete(p.launchers, k)
				}()
				p.mu.Unlock()
			}
		} else {
			p.mu.Unlock()
		}
	}
	if errs.Len() > 0 {
		return &errs
	}
	return nil
}

func (p *Launchpad) GetController(name string) (controller types.Controller, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if l, ok := p.launchers[name]; ok {
		return l.Controller, nil
	}
	return nil, xerrors.Errorf("fail to get controller %q: %w", name, ControllerNotFoundErr)
}

func (p *Launchpad) Close() {
	p.SetLaunchers(map[string]types.CmdParam{})
}

func changed(p1, p2 types.CmdParam) bool {
	if p1.Path != p2.Path {
		return true
	}

	if len(p1.Envs) != len(p2.Envs) {
		return true
	}

	for _, v1 := range p1.Envs {
		found := false
		for _, v2 := range p2.Envs {
			if v1 == v2 {
				found = true
			}
		}
		if !found {
			return true
		}
	}
	return false
}
