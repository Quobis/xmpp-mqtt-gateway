/*
This file defines the main behaviour of the gateway
*/

package main

import (
	"fmt"
	"log"
	"os"

	"strings"
	"time"

	"github.com/BurntSushi/toml"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	xco "github.com/sheenobu/go-xco"
	//p "xmpp-gateway/providers"
)

type StaticConfig struct {
	config      Config
	sippoClient *SippoClient

	//	provider  p.Provider
	xmppComponent Component
	mqttClient    mqttClient

	rxMqttCh chan *mqtt.Message
	rxXmppCh chan *xco.Message

	//mqttMessageStack map[string][]string
	//xmppMessageStack map[*xco.Address]string

	xmppMessageStack []xmppPair
	mqttMessageStack []mqttPair
}

type xmppPair struct {
	from  *xco.Address
	topic string
}

type mqttPair struct {
	topic   string
	content string
}

// type xmppMessageQueue struct {

// 	from	*xco.Address
// 	topic	string

// 	func get

// }

// ANTON the code inside this func should be in a goroutine
func (sc *StaticConfig) processStanza(stanza *xco.Message) error {

	topic := "/smartgrid/"

	if len(stanza.Body) == 0 {
		fmt.Printf("Not processing empty stanza! ")
		return nil
	}

	body_raw := strings.Split(stanza.Body, " ")
	switch body_raw[1] { //the second word makes the difference

	//GET variable1 from device1
	case "variable":

		topic = topic + body_raw[3] + "/" + body_raw[1]

	//GET devices
	case "devices":

		topic = topic + "listOfDevices"

	//GET values device1
	case "values":

		topic = topic + body_raw[2]
	}

	if !sc.checkSubscribed(xmppPair{stanza.From, topic}) {

		//storing message on the slice
		sc.xmppMessageStack = append(sc.xmppMessageStack, xmppPair{stanza.From, topic})

		return sc.mqttClient.mqttSub(topic)

	}

	//if not subscribed, xmpp client get response from the already subscribed topic

	xgwAdr := &xco.Address{
		LocalPart:  sc.config.Xmpp.Host,
		DomainPart: sc.config.Xmpp.Name,
	}

	returnStanza := sc.xmppComponent.createStanza(xgwAdr, stanza.From, sc.getMessage(topic))

	return sc.xmppComponent.xmppComponent.Send(returnStanza)

}

func (sc *StaticConfig) mqttToStanza(message *mqtt.Message) error {

	topic := (*message).Topic()

	err := sc.mqttClient.mqttPublish(string((*message).Payload()), (*message).Topic())

	//appending the message to the mqttStack
	sc.mqttMessageStack = append(sc.mqttMessageStack, mqttPair{(*message).Topic(), string((*message).Payload())})

	xgwAdr := &xco.Address{
		LocalPart:  sc.config.Xmpp.Host,
		DomainPart: sc.config.Xmpp.Name,
	}

	//getting all the addresses subscribed to a topic and creating stanzas to answer back
	for _, value := range sc.getAddresses(topic) {

		stanza := sc.xmppComponent.createStanza(xgwAdr, value, string((*message).Payload()))

		err = sc.xmppComponent.xmppComponent.Send(stanza)

	}

	return err
}

func main() {

	var config Config

	_, err := toml.DecodeFile(os.Args[1], &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't read config file '%s': %s\n", os.Args[1], err)
		os.Exit(1)
	}
	sc := &StaticConfig{config: config}
	sc.sippoClient, err = sc.setSippoServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Wrong configuration: %s\n", err)
		os.Exit(1)
	}

	//sc.rxHttpCh = make(chan p.RxHttp)
	sc.rxXmppCh = make(chan *xco.Message)
	sc.rxMqttCh = make(chan *mqtt.Message)

	// start goroutines
	gatewayDead := sc.runGatewayProcess()
	xmppDead := sc.runXmppProcess()
	mqttDead := sc.runMqttProcess()

	//sc.mqttClient.mqttSub("example")

	// httpDead := sc.runHttpProcess()

	// xmensaje := &xco.Message{
	// 	XMLName: xml.Name{
	// 		Local: "message",
	// 		Space: "jabber:component:accept",
	// 	},

	// 	Header: xco.Header{
	// 		From: &xco.Address{
	// 			LocalPart:  "user",
	// 			DomainPart: "domain",
	// 		},
	// 		To: &xco.Address{
	// 			LocalPart:  "222222222",
	// 			DomainPart: "wa.quobis",
	// 		},
	// 		ID: NewId(),
	// 	},
	// 	Type: "chat",
	// 	Body: "Hello world"}

	// if sc.processStanza(xmensaje) == nil {
	// 	fmt.Println("Empty message or bad formatting")
	// }

	// fmt.Printf("\nsc.xmppComponent.Name: %s", sc.xmppComponent.Name)
	// fmt.Printf("\nsc.mqttClient.Username: %s", sc.mqttClient.Username)
	// fmt.Printf("\n*xmensaje.From: %s", *xmensaje.From)
	// fmt.Printf("\nsc.config.Mqtt.ClientID: %s", sc.config.Mqtt.ClientID)
	// fmt.Printf("\nsc.config.Mqtt.Broker: %s", sc.config.Mqtt.Broker)
	// fmt.Printf("\nsc.config.Mqtt.Port: %d", sc.config.Mqtt.Port)
	// fmt.Printf("\nsc.config.Mqtt.Username: %s", sc.config.Mqtt.Username)

	for {
		select {
		//case _ = <-gatewayDead: ANTON
		case <-gatewayDead:
			log.Printf("Gateway died. Restarting")
			gatewayDead = sc.runGatewayProcess()
		case <-mqttDead:
			log.Printf("MQTT client died. Restarting")
			mqttDead = sc.runMqttProcess()
		case <-xmppDead:
			log.Printf("XMPP died. Restarting")
			time.Sleep(1 * time.Second)
			xmppDead = sc.runXmppProcess()

		}
	}
}
func (sc *StaticConfig) setSippoServer() (*SippoClient, error) {

	if sc.config.SippoServer != nil {
		auth := &Auth{
			GrantType: "password",
			Username:  sc.config.SippoServer.User,
			Password:  sc.config.SippoServer.Password,
		}
		ss := &SippoClient{
			Host: sc.config.SippoServer.Host,
			Auth: *auth,
		}
		return ss, nil
	}
	return nil, errors.New("Need to configure Sippo Server")
}

func (sc *StaticConfig) runGatewayProcess() <-chan struct{} {
	healthCh := make(chan struct{})

	go func(rxXmppCh <-chan *xco.Message, rxMqttCh <-chan *mqtt.Message) {
		defer func() {
			recover()
			close(healthCh)
		}()

		for {

			select {
			case rxXmpp := <-rxXmppCh:
				log.Println("Xmpp stanza received: ", rxXmpp.Body)

				err := sc.processStanza(rxXmpp)
				if err != nil {
					log.Printf("Error receiving xmpp msg: %s", err)
				}

			case rxMqtt := <-rxMqttCh:
				log.Println("MQTT message received: ", *rxMqtt) //ANTON
				log.Println("MQTT message received with topic: ", (*rxMqtt).Topic())
				log.Println("MQTT message received with payload: ", (*rxMqtt).Payload())

				err := sc.mqttToStanza(rxMqtt)

				if err != nil {
					log.Printf("Error receiving mqtt msg: %s", err)
				}

			}
			log.Println("gateway looping")
		}
	}(sc.rxXmppCh, sc.rxMqttCh)

	return healthCh
}

func (sc *StaticConfig) runXmppProcess() <-chan struct{} {
	c := &Component{
		Name:      sc.config.Xmpp.Name,
		Secret:    sc.config.Xmpp.Secret,
		Address:   fmt.Sprintf("%s:%d", sc.config.Xmpp.Host, sc.config.Xmpp.Port),
		gatewayRx: sc.rxXmppCh,
	}
	fmt.Println("Starting XMPP client process")
	return c.runXmppComponent(sc)
}

func (sc *StaticConfig) runMqttProcess() <-chan struct{} {
	c := &mqttClient{
		Url:       fmt.Sprintf("%s:%d", sc.config.Mqtt.Broker, sc.config.Mqtt.Port),
		Password:  sc.config.Mqtt.Password,
		Username:  sc.config.Mqtt.Username,
		ClientID:  sc.config.Mqtt.ClientID,
		gatewayRx: sc.rxMqttCh,
	}
	fmt.Println("Starting MQTT client process")
	return c.runMqttClient(sc)
}

func (sc *StaticConfig) checkSubscribed(pair xmppPair) bool {

	for _, value := range sc.xmppMessageStack {

		if value.topic == pair.topic && value.from == pair.from {
			return true
		}
	}
	return false
}

func (sc *StaticConfig) getAddresses(topic string) []*xco.Address {

	addresses := []*xco.Address{}

	for _, value := range sc.xmppMessageStack {

		if value.topic == topic {
			addresses = append(addresses, value.from)
		}
	}
	return addresses
}

func (sc *StaticConfig) getMessage(topic string) string {

	for _, value := range sc.mqttMessageStack {

		if value.topic == topic {
			return value.content
		}
	}
	return ""
}
