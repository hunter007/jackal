/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0012

import (
	"strconv"
	"time"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 1024

const lastActivityNamespace = "jabber:iq:last"

// LastActivity represents a last activity stream module.
type LastActivity struct {
	startTime  time.Time
	actorCh    chan func()
	shutdownCh <-chan struct{}
}

// New returns a last activity IQ handler module.
func New(disco *xep0030.DiscoInfo, shutdownCh <-chan struct{}) *LastActivity {
	x := &LastActivity{
		startTime:  time.Now(),
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: shutdownCh,
	}
	go x.loop()
	disco.RegisterServerFeature(lastActivityNamespace)
	disco.RegisterAccountFeature(lastActivityNamespace)
	return x
}

// MatchesIQ returns whether or not an IQ should be
// processed by the last activity module.
func (x *LastActivity) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", lastActivityNamespace) != nil
}

// ProcessIQ processes a last activity IQ taking according actions
// over the associated stream.
func (x *LastActivity) ProcessIQ(iq *xmpp.IQ) {
	x.actorCh <- func() { x.processIQ(iq) }
}

func (x *LastActivity) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.shutdownCh:
			return
		}
	}
}

func (x *LastActivity) processIQ(iq *xmpp.IQ) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if toJID.IsServer() {
		x.sendServerUptime(iq)
	} else if toJID.IsBare() {
		if x.isSubscribedTo(toJID, fromJID) {
			x.sendUserLastActivity(iq, toJID)
		} else {
			router.Route(iq.ForbiddenError())
		}
	}
}

func (x *LastActivity) sendServerUptime(iq *xmpp.IQ) {
	secs := int(time.Duration(time.Now().UnixNano()-x.startTime.UnixNano()) / time.Second)
	x.sendReply(iq, secs, "")
}

func (x *LastActivity) sendUserLastActivity(iq *xmpp.IQ, to *jid.JID) {
	if len(router.UserStreams(to.Node())) > 0 { // user online
		x.sendReply(iq, 0, "")
		return
	}
	usr, err := storage.Instance().FetchUser(to.Node())
	if err != nil {
		log.Error(err)
		router.Route(iq.InternalServerError())
		return
	}
	if usr == nil {
		router.Route(iq.ItemNotFoundError())
		return
	}
	var secs int
	var status string
	if p := usr.LastPresence; p != nil {
		secs = int(time.Duration(time.Now().UnixNano()-usr.LastPresenceAt.UnixNano()) / time.Second)
		if st := p.Elements().Child("status"); st != nil {
			status = st.Text()
		}
	}
	x.sendReply(iq, secs, status)
}

func (x *LastActivity) sendReply(iq *xmpp.IQ, secs int, status string) {
	q := xmpp.NewElementNamespace("query", lastActivityNamespace)
	q.SetText(status)
	q.SetAttribute("seconds", strconv.Itoa(secs))
	res := iq.ResultIQ()
	res.AppendElement(q)
	router.Route(res)
}

func (x *LastActivity) isSubscribedTo(contact *jid.JID, userJID *jid.JID) bool {
	if contact.Matches(userJID, jid.MatchesBare) {
		return true
	}
	ri, err := storage.Instance().FetchRosterItem(userJID.Node(), contact.ToBareJID().String())
	if err != nil {
		log.Error(err)
		return false
	}
	if ri == nil {
		return false
	}
	return ri.Subscription == rostermodel.SubscriptionTo || ri.Subscription == rostermodel.SubscriptionBoth
}
