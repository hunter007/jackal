/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"sync"

	"github.com/ortuman/jackal/module/xep0012"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/module/xep0049"
	"github.com/ortuman/jackal/module/xep0054"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/xmpp"
)

// Module represents a generic XMPP module.
type Module interface {
}

// IQHandler represents an IQ handler module.
type IQHandler interface {
	Module

	// MatchesIQ returns whether or not an IQ should be
	// processed by the module.
	MatchesIQ(iq *xmpp.IQ) bool

	// ProcessIQ processes a module IQ taking according actions
	// over the associated stream.
	ProcessIQ(iq *xmpp.IQ)
}

type Mods struct {
	LastActivity *xep0012.LastActivity
	Private      *xep0049.Private
	DiscoInfo    *xep0030.DiscoInfo
	VCard        *xep0054.VCard
	Version      *xep0092.Version

	iqHandlers []IQHandler
	all        []Module
}

var (
	instMu      sync.RWMutex
	mods        Mods
	shutdownCh  chan struct{}
	initialized bool
)

func Initialize(cfg *Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	initializeModules(cfg)
	initialized = true
}

func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	if !initialized {
		return
	}
	close(shutdownCh)
	mods = Mods{}
	initialized = false
}

func Modules() Mods {
	return mods
}

func IQHandlers() []IQHandler {
	return mods.iqHandlers
}

func initializeModules(cfg *Config) {
	shutdownCh = make(chan struct{})
	mods.DiscoInfo = xep0030.New(shutdownCh)
	mods.iqHandlers = append(mods.iqHandlers, mods.DiscoInfo)
	mods.all = append(mods.all, mods.DiscoInfo)

	// XEP-0012: Last Activity (https://xmpp.org/extensions/xep-0012.html)
	if _, ok := cfg.Enabled["last_activity"]; ok {
		mods.LastActivity = xep0012.New(mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.LastActivity)
		mods.all = append(mods.all, mods.LastActivity)
	}

	// XEP-0049: Private XML Storage (https://xmpp.org/extensions/xep-0049.html)
	if _, ok := cfg.Enabled["private"]; ok {
		mods.Private = xep0049.New(shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.Private)
		mods.all = append(mods.all, mods.Private)
	}

	// XEP-0054: vcard-temp (https://xmpp.org/extensions/xep-0054.html)
	if _, ok := cfg.Enabled["vcard"]; ok {
		mods.VCard = xep0054.New(mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.VCard)
		mods.all = append(mods.all, mods.VCard)
	}

	// XEP-0092: Software Version (https://xmpp.org/extensions/xep-0092.html)
	if _, ok := cfg.Enabled["version"]; ok {
		mods.Version = xep0092.New(&cfg.Version, mods.DiscoInfo, shutdownCh)
		mods.iqHandlers = append(mods.iqHandlers, mods.Version)
		mods.all = append(mods.all, mods.Version)
	}
}
