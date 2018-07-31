/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"fmt"
	"sync"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

// DiscoInfo represents a disco info server stream module.
type DiscoInfo struct {
	stm      stream.C2S
	mu       sync.RWMutex
	entities map[string]*Entity
}

// New returns a disco info IQ handler module.
func New(stm stream.C2S) *DiscoInfo {
	return &DiscoInfo{
		stm:      stm,
		entities: make(map[string]*Entity),
	}
}

// RegisterDisco registers disco entity features/items
// associated to disco info module.
func (di *DiscoInfo) RegisterDisco(discoInfo *DiscoInfo) {
}

// RegisterEntity registers a new disco entity associated to a jid
// and an optional node.
func (di *DiscoInfo) RegisterEntity(jid, node string) (*Entity, error) {
	k := di.entityKey(jid, node)
	di.mu.Lock()
	defer di.mu.Unlock()
	if _, ok := di.entities[k]; ok {
		return nil, fmt.Errorf("entity already registered: %s", k)
	}
	ent := &Entity{}
	ent.AddFeature(discoInfoNamespace)
	ent.AddFeature(discoItemsNamespace)
	di.entities[k] = ent
	return ent, nil
}

// Entity returns a previously registered disco entity.
func (di *DiscoInfo) Entity(jid, node string) *Entity {
	k := di.entityKey(jid, node)
	di.mu.RLock()
	e := di.entities[k]
	di.mu.RUnlock()
	return e
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
	q := iq.Elements().Child("query")
	ent := di.Entity(iq.ToJID().String(), q.Attributes().Get("node"))
	if ent == nil {
		di.stm.SendElement(iq.ItemNotFoundError())
		return
	}
	switch q.Namespace() {
	case discoInfoNamespace:
		di.sendDiscoInfo(ent, iq)
	case discoItemsNamespace:
		di.sendDiscoItems(ent, iq)
	}
}

func (di *DiscoInfo) sendDiscoInfo(ent *Entity, iq *xmpp.IQ) {
	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", discoInfoNamespace)

	for _, identity := range ent.Identities() {
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
	for _, feature := range ent.Features() {
		featureEl := xmpp.NewElementName("feature")
		featureEl.SetAttribute("var", feature)
		query.AppendElement(featureEl)
	}
	result.AppendElement(query)
	di.stm.SendElement(result)
}

func (di *DiscoInfo) sendDiscoItems(ent *Entity, iq *xmpp.IQ) {
	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", discoItemsNamespace)

	for _, item := range ent.Items() {
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
	di.stm.SendElement(result)
}

func (di *DiscoInfo) entityKey(jid, node string) string {
	k := jid
	if len(node) > 0 {
		k += ":"
		k += node
	}
	return k
}
