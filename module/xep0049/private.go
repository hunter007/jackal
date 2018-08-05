/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0049

import (
	"strings"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 1024

const privateNamespace = "jabber:iq:private"

// Private represents a private storage server stream module.
type Private struct {
	actorCh    chan func()
	shutdownCh <-chan struct{}
}

// New returns a private storage IQ handler module.
func New(shutdownCh <-chan struct{}) *Private {
	x := &Private{
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: shutdownCh,
	}
	go x.loop()
	return x
}

// MatchesIQ returns whether or not an IQ should be
// processed by the private storage module.
func (x *Private) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.Elements().ChildNamespace("query", privateNamespace) != nil
}

// ProcessIQ processes a private storage IQ taking according actions
// over the associated stream.
func (x *Private) ProcessIQ(iq *xmpp.IQ) {
	x.actorCh <- func() { x.processIQ(iq) }
}

func (x *Private) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.shutdownCh:
			return
		}
	}
}

func (x *Private) processIQ(iq *xmpp.IQ) {
	userJID := iq.FromJID()
	toJID := iq.ToJID()
	q := iq.Elements().ChildNamespace("query", privateNamespace)
	validTo := toJID.IsServer() || toJID.Node() == userJID.Node()
	if !validTo {
		router.Route(iq.ForbiddenError())
		return
	}
	if iq.IsGet() {
		x.getPrivate(userJID, iq, q)
	} else if iq.IsSet() {
		x.setPrivate(userJID, iq, q)
	} else {
		router.Route(iq.BadRequestError())
		return
	}
}

func (x *Private) getPrivate(userJID *jid.JID, iq *xmpp.IQ, q xmpp.XElement) {
	if q.Elements().Count() != 1 {
		router.Route(iq.NotAcceptableError())
		return
	}
	privElem := q.Elements().All()[0]
	privNS := privElem.Namespace()
	isValidNS := x.isValidNamespace(privNS)

	if privElem.Elements().Count() > 0 || !isValidNS {
		router.Route(iq.NotAcceptableError())
		return
	}
	log.Infof("retrieving private element. ns: %s... (%s/%s)", privNS, userJID.Node(), userJID.Resource())

	privElements, err := storage.Instance().FetchPrivateXML(privNS, userJID.Node())
	if err != nil {
		log.Errorf("%v", err)
		router.Route(iq.InternalServerError())
		return
	}
	res := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", privateNamespace)
	if privElements != nil {
		query.AppendElements(privElements)
	} else {
		query.AppendElement(xmpp.NewElementNamespace(privElem.Name(), privElem.Namespace()))
	}
	res.AppendElement(query)
	router.Route(res)
}

func (x *Private) setPrivate(userJID *jid.JID, iq *xmpp.IQ, q xmpp.XElement) {
	nsElements := map[string][]xmpp.XElement{}

	for _, privElement := range q.Elements().All() {
		ns := privElement.Namespace()
		if len(ns) == 0 {
			router.Route(iq.BadRequestError())
			return
		}
		if !x.isValidNamespace(privElement.Namespace()) {
			router.Route(iq.NotAcceptableError())
			return
		}
		elems := nsElements[ns]
		if elems == nil {
			elems = []xmpp.XElement{privElement}
		} else {
			elems = append(elems, privElement)
		}
		nsElements[ns] = elems
	}
	for ns, elements := range nsElements {
		log.Infof("saving private element. ns: %s... (%s/%s)", ns, userJID.Node(), userJID.Resource())

		if err := storage.Instance().InsertOrUpdatePrivateXML(elements, ns, userJID.Node()); err != nil {
			log.Error(err)
			router.Route(iq.InternalServerError())
			return
		}
	}
	router.Route(iq.ResultIQ())
}

func (x *Private) isValidNamespace(ns string) bool {
	return !strings.HasPrefix(ns, "jabber:") && !strings.HasPrefix(ns, "http://jabber.org/") && ns != "vcard-temp"
}
