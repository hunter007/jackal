/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"testing"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0030_Matching(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(nil)

	// test MatchesIQ
	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)

	require.False(t, x.MatchesIQ(iq1))

	iq1.AppendElement(xmpp.NewElementNamespace("query", discoItemsNamespace))

	iq2 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq2.SetFromJID(j)
	iq2.AppendElement(xmpp.NewElementNamespace("query", discoItemsNamespace))

	require.True(t, x.MatchesIQ(iq1))
	require.True(t, x.MatchesIQ(iq2))

	iq1.SetType(xmpp.SetType)
	iq2.SetType(xmpp.ResultType)

	require.False(t, x.MatchesIQ(iq1))
	require.False(t, x.MatchesIQ(iq2))
}

func TestXEP0030_SetItems(t *testing.T) {
	x := New(nil)
	x.RegisterEntity("jackal.im", "")

	its := []Item{
		{Jid: "j1@jackal.im", Name: "a name", Node: "node1"},
		{Jid: "j2@jackal.im", Name: "a second name", Node: "node2"},
	}
	ent := x.Entity("jackal.im", "")
	ent.AddItem(its[0])
	ent.AddItem(its[1])

	require.Equal(t, its, ent.Items())
}

func TestXEP0030_SetIdentities(t *testing.T) {
	x := New(nil)
	x.RegisterEntity("jackal.im", "")

	ids := []Identity{{
		Category: "server",
		Type:     "im",
		Name:     "default",
	}}
	ent := x.Entity("jackal.im", "")
	ent.AddIdentity(ids[0])

	require.Equal(t, ids, ent.Identities())
}

func TestXEP0030_SetFeatures(t *testing.T) {
	x := New(nil)
	x.RegisterEntity("jackal.im", "")

	fs := []Feature{
		discoInfoNamespace,
		discoItemsNamespace,
	}
	ent := x.Entity("jackal.im", "")
	require.Equal(t, fs, ent.Features())
}

func TestXEP0030_BadToJID(t *testing.T) {
	j, _ := jid.New("", "example.im", "", true)
	stm := stream.NewMockC2S("abcd", j)

	x := New(stm)
	x.RegisterEntity("jackal.im", "")

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xmpp.NewElementNamespace("query", discoItemsNamespace))

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	require.Equal(t, xmpp.ErrItemNotFound.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0030_GetFeatures(t *testing.T) {
	srvJid, _ := jid.New("", "jackal.im", "", true)

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	stm := stream.NewMockC2S("abcd", j)

	x := New(stm)
	x.RegisterEntity("jackal.im", "")

	ent := x.Entity("jackal.im", "")
	ent.AddIdentity(Identity{
		Category: "server",
		Type:     "im",
		Name:     "default",
	})

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(srvJid)
	iq1.AppendElement(xmpp.NewElementNamespace("query", discoInfoNamespace))

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	q := elem.Elements().ChildNamespace("query", discoInfoNamespace)
	require.Equal(t, 3, q.Elements().Count())
	require.Equal(t, "identity", q.Elements().All()[0].Name())
	require.Equal(t, "feature", q.Elements().All()[1].Name())
}

func TestXEP0030_GetItems(t *testing.T) {
	srvJid, _ := jid.New("", "jackal.im", "", true)

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	stm := stream.NewMockC2S("abcd", j)

	x := New(stm)
	x.RegisterEntity("jackal.im", "http://jabber.org/protocol/commands")

	ent := x.Entity("jackal.im", "http://jabber.org/protocol/commands")
	ent.AddItem(Item{Jid: "j1@jackal.im", Name: "a name", Node: "node1"})
	ent.AddItem(Item{Jid: "j2@jackal.im", Name: "a second name", Node: "node2"})

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(srvJid)
	q := xmpp.NewElementNamespace("query", discoItemsNamespace)
	q.SetAttribute("node", "http://jabber.org/protocol/commands")
	iq1.AppendElement(q)

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	q2 := elem.Elements().ChildNamespace("query", discoItemsNamespace)
	require.Equal(t, 2, q2.Elements().Count())
	require.Equal(t, "item", q2.Elements().All()[0].Name())
}
