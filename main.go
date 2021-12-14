/*
This file defines the main behaviour of the gateway
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"time"

	"strings"

	"github.com/BurntSushi/toml"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	xco "github.com/sheenobu/go-xco"
	//p "xmpp-gateway/providers"
)

type StaticConfig struct {
	config Config
	//sippoClient *SippoClient

	//	provider  p.Provider
	xmppComponent Component
	mqttClient    mqttClient

	rxMqttCh chan *mqtt.Message
	rxXmppCh chan *xco.Message

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

type Mensajee struct {
	Variable string `json:variable`
	Value    string `json:value`
}

func main() {

	var config Config

	_, err := toml.DecodeFile(os.Args[1], &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't read config file '%s': %s\n", os.Args[1], err)
		os.Exit(1)
	}
	sc := &StaticConfig{config: config}
	//sc.sippoClient, err = sc.setSippoServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Wrong configuration: %s\n", err)
		os.Exit(1)
	}

	//sc.rxHttpCh = make(chan p.RxHttp)
	sc.rxXmppCh = make(chan *xco.Message)
	sc.rxMqttCh = make(chan *mqtt.Message)

	//Testing the subscribe to a topic before publish (not persistent message stack)

	// s := mqttPair{
	// 	"/smartgrid/listOfDevices",
	// 	"deviceListExample",
	// }
	// sc.mqttMessageStack = append(sc.mqttMessageStack, s)

	// start goroutines
	gatewayDead := sc.runGatewayProcess()
	xmppDead := sc.runXmppProcess()
	mqttDead := sc.runMqttProcess()

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
				log.Println("MQTT message received with ID: ", (*rxMqtt).MessageID()) //ANTON
				log.Println("MQTT message received with topic: ", (*rxMqtt).Topic())
				log.Println("MQTT message received with payload: ", string((*rxMqtt).Payload()))

				err := sc.mqttToStanza(rxMqtt)

				if err != nil {
					log.Printf("Error receiving mqtt msg: %s", err)
				}

			}
			log.Println("gateway looping \n")
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

// ANTON the code inside this func should be in a goroutine
func (sc *StaticConfig) processStanza(stanza *xco.Message) error {

	topic := "/smartgrid/"

	xgwAdr := &xco.Address{
		DomainPart: sc.config.Xmpp.Name,
	}

	if len(stanza.Body) == 0 {
		fmt.Printf("Not processing empty stanza! ")
		return nil
	}
	body_raw := strings.Split(stanza.Body, " ")

	//if the message is not a get, ignore it
	if strings.Compare(body_raw[0], "GET") == 0 {

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

			err := sc.mqttClient.mqttSub(topic)

			// for _, value := range sc.mqttMessageStack {
			// 	fmt.Printf("Mensajes en la pila: Topic: %s, Message: %s\n", value.topic, value.content)
			// }

			messagePublished := sc.getMessage(topic)

			fmt.Print(messagePublished)

			if messagePublished != "" {

				fmt.Printf("Sending message: %s, from %s, to %s:%s \n", messagePublished, xgwAdr.DomainPart, stanza.From.LocalPart, stanza.From.DomainPart)

				returnStanza := sc.xmppComponent.createStanza(xgwAdr, stanza.From, messagePublished)
				return sc.xmppComponent.xmppComponent.Send(returnStanza)

			}

			return err
		}

		//if subscribed, xmpp client get response from the already subscribed topic

		returnStanza := sc.xmppComponent.createStanza(xgwAdr, stanza.From, sc.getMessage(topic))
		return sc.xmppComponent.xmppComponent.Send(returnStanza)

	}

	//not standard message, ignoring it
	return errors.New("Wrong xmpp body format.")
}

func (sc *StaticConfig) mqttToStanza(message *mqtt.Message) error {

	topic := (*message).Topic()
	err := errors.New("")
	var text string

	switch topic {

	case "/smartgrid/listOfDevices":

		//`["1","2","3"]`

		dataJson := (*message).Payload()

		var devices []string

		err = json.Unmarshal([]byte(dataJson), &devices)

		text += "Devices: \n"

		for i, value := range devices {
			text = text + strconv.Itoa(i) + ".- " + value + "\n"
		}

	case "/smartgrid/device1/variable1":

		//`{"variable": "variable1", "value": "value1"}`

		var m Mensajee
		err = json.Unmarshal((*message).Payload(), &m)

		if err != nil {
			log.Fatal(err)
		}

		body_raw := strings.Split(topic, "/")

		fmt.Printf("Getting %s from %s:\n", body_raw[2], body_raw[1])

		fmt.Printf("%s: %s\n", m.Variable, m.Value)

	case "/smartgrid/device1":

		//`[{"variable": "variable1", "value": "value1"}, {"variable": "variable2", "value": "value2"}]`

		var m []Mensajee

		err = json.Unmarshal((*message).Payload(), &m)

		if err != nil {
			log.Fatal(err)
		}

		body_raw := strings.Split(topic, "/")

		fmt.Printf("Getting %s from %s:\n", body_raw[2], body_raw[1])

		for _, value := range m {

			fmt.Printf("%s: %s\n", value.Variable, value.Value)
		}
	}

	//appending the message to the mqttStack
	sc.mqttMessageStack = append(sc.mqttMessageStack, mqttPair{(*message).Topic(), text})

	xgwAdr := &xco.Address{
		//LocalPart:  sc.config.Xmpp.Host,
		DomainPart: sc.config.Xmpp.Name,
	}

	//getting all the addresses subscribed to a topic and creating stanzas to answer back
	for _, value := range sc.getAddresses(topic) {

		stanza := sc.xmppComponent.createStanza(xgwAdr, value, text)
		fmt.Printf("Stanza to be send: Body: %s; From: %s\n", text, value.DomainPart)

		err = sc.xmppComponent.xmppComponent.Send(stanza)

	}

	return err

}

//checks if xmpp client is subscribed to topic based on xmppmessage "stack"
func (sc *StaticConfig) checkSubscribed(pair xmppPair) bool {

	for _, value := range sc.xmppMessageStack {
		if value.topic == pair.topic && reflect.DeepEqual(value.from, pair.from) {
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

// func (sc *StaticConfig) setSippoServer() (*SippoClient, error) {

// 	if sc.config.SippoServer != nil {
// 		auth := &Auth{
// 			GrantType: "password",
// 			Username:  sc.config.SippoServer.User,
// 			Password:  sc.config.SippoServer.Password,
// 		}
// 		ss := &SippoClient{
// 			Host: sc.config.SippoServer.Host,
// 			Auth: *auth,
// 		}
// 		return ss, nil
// 	}
// 	return nil, errors.New("Need to configure Sippo Server")
// }
