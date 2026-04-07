package domainplugin

import "fmt"

type Registry struct {
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

func (r *Registry) Register(p Plugin) {
	r.plugins[p.Name()] = p
}

func (r *Registry) Get(domain string) (Plugin, error) {
	p, ok := r.plugins[domain]
	if !ok {
		return nil, fmt.Errorf("domain plugin not found: %s", domain)
	}
	return p, nil
}

func (r *Registry) All() map[string]Plugin {
	return r.plugins
}
