package plugin

import (
	"strings"
	"sync"
)

type Errors []error

func (e Errors) Error() string {
	m := make([]string, len(e))
	for i, err := range e {
		m[i] = err.Error()
	}
	return strings.Join(m, " ")
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

func (p *Launchpad) SetLaunchers(params map[string]CmdParam) error {
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
				errs = append(errs, err)
			} else {
				p.mu.Lock()
				p.launchers[k] = launcher
				p.mu.Unlock()
			}
		} else {
			p.mu.Unlock()
		}
	}
	return errs
}

func (p *Launchpad) GetController(name string) Controller {
	p.mu.Lock()
	defer p.mu.Unlock()
	if l, ok := p.launchers[name]; ok {
		return l.Controller
	}
	return nil
}

func (p *Launchpad) Close() {
	p.SetLaunchers(map[string]CmdParam{})
}

func changed(p1, p2 CmdParam) bool {
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
