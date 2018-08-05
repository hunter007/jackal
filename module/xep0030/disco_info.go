/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"sync"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 4096

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

// Feature represents a disco info feature entity.
type Feature = string

// Identity represents a disco info identity entity.
type Identity struct {
	Category string
	Type     string
	Name     string
}

// Item represents a disco info item entity.
type Item struct {
	Jid  string
	Name string
	Node string
}

type Provider interface {
	Identities(toJID, fromJID *jid.JID, node string) []Identity
	Items(toJID, fromJID *jid.JID, node string) ([]Item, *xmpp.StanzaError)
	Features(toJID, fromJID *jid.JID, node string) ([]Feature, *xmpp.StanzaError)
}

// DiscoInfo represents a disco info server stream module.
type DiscoInfo struct {
	mu          sync.RWMutex
	actorCh     chan func()
	shutdownCh  <-chan struct{}
	srvProvider *serverProvider
	providers   map[string]Provider
}

// New returns a disco info IQ handler module.
func New(shutdownCh <-chan struct{}) *DiscoInfo {
	di := &DiscoInfo{
		srvProvider: &serverProvider{},
		providers:   make(map[string]Provider),
		actorCh:     make(chan func(), mailboxSize),
		shutdownCh:  shutdownCh,
	}
	go di.loop()
	di.RegisterServerFeature(discoItemsNamespace)
	di.RegisterServerFeature(discoInfoNamespace)
	di.RegisterAccountFeature(discoItemsNamespace)
	di.RegisterAccountFeature(discoInfoNamespace)
	return di
}

func (di *DiscoInfo) RegisterServerFeature(feature string) {
	di.srvProvider.registerServerFeature(feature)
}

func (di *DiscoInfo) UnregisterServerFeature(feature string) {
	di.srvProvider.unregisterServerFeature(feature)
}

func (di *DiscoInfo) RegisterAccountFeature(feature string) {
	di.srvProvider.registerAccountFeature(feature)
}

func (di *DiscoInfo) UnregisterAccountFeature(feature string) {
	di.srvProvider.unregisterAccountFeature(feature)
}

func (di *DiscoInfo) RegisterProvider(domain string, provider Provider) {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.providers[domain] = provider
}

func (di *DiscoInfo) UnregisterProvider(domain string) {
	di.mu.Lock()
	defer di.mu.Unlock()
	delete(di.providers, domain)
}

// MatchesIQ returns whether or not an IQ should be
// processed by the disco info module.
func (di *DiscoInfo) MatchesIQ(iq *xmpp.IQ) bool {
	q := iq.Elements().Child("query")
	if q == nil {
		return false
	}
	return iq.IsGet() && (q.Namespace() == discoInfoNamespace || q.Namespace() == discoItemsNamespace)
}

// ProcessIQ processes a disco info IQ taking according actions
// over the associated stream.
func (di *DiscoInfo) ProcessIQ(iq *xmpp.IQ) {
	di.actorCh <- func() { di.processIQ(iq) }
}

func (di *DiscoInfo) loop() {
	for {
		select {
		case f := <-di.actorCh:
			f()
		case <-di.shutdownCh:
			return
		}
	}
}

func (di *DiscoInfo) processIQ(iq *xmpp.IQ) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	var prov Provider
	if host.IsLocalHost(toJID.Domain()) {
		prov = di.srvProvider
	} else {
		prov = di.providers[toJID.Domain()]
		if prov == nil {
			router.Route(iq.ItemNotFoundError())
			return
		}
	}
	q := iq.Elements().Child("query")
	node := q.Attributes().Get("node")
	switch q.Namespace() {
	case discoInfoNamespace:
		di.sendDiscoInfo(prov, toJID, fromJID, node, iq)
	case discoItemsNamespace:
		di.sendDiscoItems(prov, toJID, fromJID, node, iq)
	}
}

func (di *DiscoInfo) sendDiscoInfo(prov Provider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ) {
	features, sErr := prov.Features(toJID, fromJID, node)
	if sErr != nil {
		router.Route(xmpp.NewErrorElementFromElement(iq, sErr, nil))
		return
	} else if len(features) == 0 {
		router.Route(iq.ItemNotFoundError())
		return
	}
	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", discoInfoNamespace)

	identities := prov.Identities(toJID, fromJID, node)
	for _, identity := range identities {
		identityEl := xmpp.NewElementName("identity")
		identityEl.SetAttribute("category", identity.Category)
		if len(identity.Type) > 0 {
			identityEl.SetAttribute("type", identity.Type)
		}
		if len(identity.Name) > 0 {
			identityEl.SetAttribute("name", identity.Name)
		}
		query.AppendElement(identityEl)
	}
	for _, feature := range features {
		featureEl := xmpp.NewElementName("feature")
		featureEl.SetAttribute("var", feature)
		query.AppendElement(featureEl)
	}
	result.AppendElement(query)
	router.Route(result)
}

func (di *DiscoInfo) sendDiscoItems(prov Provider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ) {
	items, sErr := prov.Items(toJID, fromJID, node)
	if sErr != nil {
		router.Route(xmpp.NewErrorElementFromElement(iq, sErr, nil))
		return
	} else if len(items) == 0 {
		router.Route(iq.ItemNotFoundError())
		return
	}
	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", discoItemsNamespace)
	for _, item := range items {
		itemEl := xmpp.NewElementName("item")
		itemEl.SetAttribute("jid", item.Jid)
		if len(item.Name) > 0 {
			itemEl.SetAttribute("name", item.Name)
		}
		if len(item.Node) > 0 {
			itemEl.SetAttribute("node", item.Node)
		}
		query.AppendElement(itemEl)
	}
	result.AppendElement(query)
	router.Route(result)
}
