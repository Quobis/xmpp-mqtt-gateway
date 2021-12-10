/*
This file contains methods for error handling, generating and dealing with stanzas,
and creating new ids for the xmpp component
*/

package main

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strings"

	xco "github.com/sheenobu/go-xco"
)

type Component struct {
	Name          string
	Secret        string
	Address       string
	xmppComponent *xco.Component
	gatewayRx     chan<- *xco.Message
}

func (c *Component) runXmppComponent(sc *StaticConfig) <-chan struct{} {

	opts := xco.Options{
		Name:         c.Name,
		SharedSecret: c.Secret,
		Address:      c.Address,
		Logger:       log.New(os.Stderr, "", log.LstdFlags),
	}

	fmt.Println("Inside XMPP client")

	healthCh := make(chan struct{})
	go func() {
		defer func() {
			recover()
			close(healthCh)
		}()

		x, err := xco.NewComponent(opts)
		if err != nil {
			log.Printf("can't create internal XMPP component: %s", err)
			return
		}

		x.MessageHandler = c.onMessage
		x.PresenceHandler = c.onPresence
		x.IqHandler = c.onIq
		x.UnknownHandler = c.onUnknown
		c.xmppComponent = x
		sc.xmppComponent = *c

		err = x.Run()

		//exemplo
		//x.Send(opts)

		log.Printf("lost XMPP connection:%s", err)
	}()
	return healthCh
}

func (c *Component) createStanza(from, to *xco.Address, body string) *xco.Message {
	msg := &xco.Message{
		XMLName: xml.Name{
			Local: "message",
			Space: "jabber:component:accept",
		},

		Header: xco.Header{
			From: from,
			To:   to,
			ID:   NewId(),
		},
		Type: "chat",
		Body: body,
	}
	return msg
}

func (c *Component) onMessage(x *xco.Component, m *xco.Message) error {
	log.Printf("Message: %+v, To: %s", m, m.To.LocalPart)
	if m.Body == "" {
		log.Printf("  ignoring message with empty body")
		return nil
	}
	go func() { c.gatewayRx <- m }()
	return nil
}

func (c *Component) onPresence(x *xco.Component, p *xco.Presence) error {
	log.Printf("Presence: %+v", p)
	return nil
}

func (c *Component) onIq(x *xco.Component, iq *xco.Iq) error {
	log.Printf("Iq: %+v", iq)
	return nil
}

func (c *Component) onUnknown(x *xco.Component, s *xml.StartElement) error {
	log.Printf("Unknown: %+v", x)
	return nil
}

func NewId() string {
	// generate 128 random bits (6 more than standard UUID)
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	// convert them to base 32 encoding
	s := base32.StdEncoding.EncodeToString(bytes)
	return strings.ToLower(strings.TrimRight(s, "="))
}
