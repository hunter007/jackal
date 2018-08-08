/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package component

import (
	"fmt"
	"sync"

	"github.com/ortuman/jackal/component/httpupload"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xmpp"
)

type Component interface {
	Host() string
	ServiceName() string
	ProcessStanza(stanza xmpp.Stanza)
	Shutdown()
}

// singleton interface
var (
	instMu      sync.RWMutex
	comps       map[string]Component
	initialized bool
)

// Initialize initializes the components manager.
func Initialize(cfg *Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	comps = make(map[string]Component)

	cs := loadComponents(cfg)
	for _, c := range cs {
		host := c.Host()
		if _, ok := comps[host]; ok {
			log.Fatalf("%v", fmt.Errorf("component host name conflict: %s", host))
		}
		comps[host] = c
	}
	initialized = true
}

// Shutdown shuts down components manager system.
// This method should be used only for testing purposes.
func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return
	}
	for _, comp := range comps {
		comp.Shutdown()
	}
	comps = nil
	initialized = false
}

func Get(host string) Component {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return nil
	}
	return comps[host]
}

func GetAll() []Component {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return nil
	}
	var ret []Component
	for _, comp := range comps {
		ret = append(ret, comp)
	}
	return ret
}

func loadComponents(cfg *Config) []Component {
	var ret []Component
	if cfg.HttpUpload != nil {
		ret = append(ret, httpupload.New(cfg.HttpUpload))
	}
	return ret
}
