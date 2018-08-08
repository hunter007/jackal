/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package httpupload

import (
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 1024

const httpUploadServiceName = "HTTP File Upload"

type HttpUpload struct {
	cfg     *Config
	actorCh chan func()
	closeCh chan struct{}
}

func New(cfg *Config) *HttpUpload {
	h := &HttpUpload{
		cfg:     cfg,
		actorCh: make(chan func(), mailboxSize),
		closeCh: make(chan struct{}),
	}
	go h.loop()
	return h
}

func (c *HttpUpload) Host() string {
	return c.cfg.Host
}

func (c *HttpUpload) ServiceName() string {
	return httpUploadServiceName
}

func (c *HttpUpload) ProcessStanza(stanza xmpp.Stanza) {
	c.actorCh <- func() {
		c.processStanza(stanza)
	}
}

func (c *HttpUpload) Shutdown() {
}

func (c *HttpUpload) loop() {
	for {
		select {
		case f := <-c.actorCh:
			f()
		case <-c.closeCh:
			return
		}
	}
}

func (c *HttpUpload) processStanza(stanza xmpp.XElement) {
}
