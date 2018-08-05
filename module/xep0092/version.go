/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0092

import (
	"os/exec"
	"strings"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 1024

const versionNamespace = "jabber:iq:version"

var osString string

func init() {
	out, _ := exec.Command("uname", "-rs").Output()
	osString = strings.TrimSpace(string(out))
}

// Config represents XMPP Software Version module (XEP-0092) configuration.
type Config struct {
	ShowOS bool `yaml:"show_os"`
}

// Version represents a version module.
type Version struct {
	cfg        *Config
	actorCh    chan func()
	shutdownCh <-chan struct{}
}

// New returns a version IQ handler module.
func New(config *Config, disco *xep0030.DiscoInfo, shutdownCh <-chan struct{}) *Version {
	v := &Version{
		cfg:        config,
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: shutdownCh,
	}
	go v.loop()
	disco.RegisterServerFeature(versionNamespace)
	return v
}

// MatchesIQ returns whether or not an IQ should be
// processed by the version module.
func (x *Version) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", versionNamespace) != nil && iq.ToJID().IsServer()
}

// ProcessIQ processes a version IQ taking according actions
// over the associated stream.
func (x *Version) ProcessIQ(iq *xmpp.IQ) {
	x.actorCh <- func() { x.processIQ(iq) }
}

func (x *Version) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.shutdownCh:
			return
		}
	}
}

func (x *Version) processIQ(iq *xmpp.IQ) {
	q := iq.Elements().ChildNamespace("query", versionNamespace)
	if q.Elements().Count() != 0 {
		router.Route(iq.BadRequestError())
		return
	}
	x.sendSoftwareVersion(iq)
}

func (x *Version) sendSoftwareVersion(iq *xmpp.IQ) {
	fromJID := iq.FromJID()
	username := fromJID.Node()
	resource := fromJID.Resource()
	log.Infof("retrieving software version: %v (%s/%s)", version.ApplicationVersion, username, resource)

	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", versionNamespace)

	name := xmpp.NewElementName("name")
	name.SetText("jackal")
	query.AppendElement(name)

	ver := xmpp.NewElementName("version")
	ver.SetText(version.ApplicationVersion.String())
	query.AppendElement(ver)

	if x.cfg.ShowOS {
		os := xmpp.NewElementName("os")
		os.SetText(osString)
		query.AppendElement(os)
	}
	result.AppendElement(query)
	router.Route(result)
}
