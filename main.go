/*
This file defines the main behaviour of the gateway
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	//"reflect"
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

	xmppMessageStack map[xco.Address][]string
	mqttMessageStack map[string]string
}

// type xmppPair struct {
// 	from  *xco.Address
// 	topic string
// }

// type mqttPair struct {
// 	topic   string
// 	content string
// }

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

	sc.xmppMessageStack = make(map[xco.Address][]string)
	sc.mqttMessageStack = make(map[string]string)

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
				log.Println("MQTT message received with payload: \n", string((*rxMqtt).Payload()))

				err := sc.mqttToStanza(rxMqtt)

				if err != nil {
					log.Printf("Error receiving mqtt msg: %s", err)
				}

				fmt.Printf("%v", sc.mqttMessageStack)

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

	topic := "smartgrid/"

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

		//GET devices
		case "devices":

			topic += "listOfDevices"

		//GET values device
		case "values":

			topic += body_raw[2]

		default:

			//GET variable1 from device1
			if strings.Compare(body_raw[2], "from") == 0 {
				topic += body_raw[3] + "/" + body_raw[1]
				break
			}

			return errors.New("Bad formatting on the topic")

		}

		if !sc.checkSubscribed(*stanza.From, topic) {
			//storing message on the array
			sc.xmppMessageStack[*stanza.From] = append(sc.xmppMessageStack[*stanza.From], topic)
			err := sc.mqttClient.mqttSub(topic)
			messagePublished := sc.mqttMessageStack[topic]

			fmt.Print(messagePublished)

			if messagePublished != "" {

				fmt.Printf("Sending message: %s, from %s, to %s:%s \n", messagePublished, xgwAdr.DomainPart, stanza.From.LocalPart, stanza.From.DomainPart)

				returnStanza := sc.xmppComponent.createStanza(xgwAdr, stanza.From, messagePublished)
				return sc.xmppComponent.xmppComponent.Send(returnStanza)

			}

			return err
		}
		//if subscribed, xmpp client get response from the already subscribed topic

		returnStanza := sc.xmppComponent.createStanza(xgwAdr, stanza.From, sc.mqttMessageStack[topic])
		return sc.xmppComponent.xmppComponent.Send(returnStanza)

	}

	//not standard message, ignoring it
	return errors.New("Wrong xmpp body format.")
}

func (sc *StaticConfig) mqttToStanza(message *mqtt.Message) error {

	topic := (*message).Topic()

	if len(topic) == 0 {
		return errors.New("Not publishing on empty topic!")
	}

	err := errors.New("")
	text := "\n"
	topic_raw := strings.Split(topic, "/")

	if strings.Compare(topic_raw[0], "smartgrid") != 0 {

		return errors.New("Bad topic on mqtt publish")
	}

	// Given a possibly complex JSON object
	msg := string((*message).Payload())

	//we could list devices based on the "type" field
	if strings.Compare(topic_raw[1], "listOfDevices") == 0 {

		var devices []string
		err = json.Unmarshal([]byte(msg), &devices)

		text += "Devices: \n"

		for _, value := range devices {
			text += "\t" + value + "\n"
		}

	} else {

		// We only know our top-level keys are strings
		mp := make(map[string]interface{})

		// Decode JSON into our map
		err := json.Unmarshal([]byte(msg), &mp)
		if err != nil {
			println(err)
			return err
		}

		//If the topic has the format (smartgrid/device_id/variable)
		if len(topic_raw) == 3 {

			//Battery || Thermostat || Solar_kit: device_id
			//Type field needs to be included and not nil
			if mp["type"] != nil {
				text += fmt.Sprintf("%v", mp["type"]) + ": " + topic_raw[1] + "\n"
			} else {
				text += "Device" + ": " + topic_raw[1] + "\n"
			}

			text += "\t" + topic_raw[2] + ": " + fmt.Sprintf("%v", mp[topic_raw[2]])

		} else {

			// Iterate the map
			// Note: that mp has to be deferenced here or range will fail
			text += topic_raw[1] + ":\n"
			for key, value := range mp {
				text += "\t" + string(key) + " : " + fmt.Sprintf("%v", value) + "\n"
			}
		}
	}
	//setting the message to the mqttStack
	sc.mqttMessageStack[topic] = text

	xgwAdr := &xco.Address{
		//LocalPart:  sc.config.Xmpp.Host,
		DomainPart: sc.config.Xmpp.Name,
	}

	//getting all the addresses subscribed to a topic and creating stanzas to answer back
	for _, value := range sc.getAddresses(topic) {

		stanza := sc.xmppComponent.createStanza(xgwAdr, &value, text)
		fmt.Printf("Stanza to be send: \nBody: %s\n; From: %s\n", text, value.DomainPart)

		err = sc.xmppComponent.xmppComponent.Send(stanza)

	}

	return err

}

//checks if xmpp client is subscribed to topic based on xmppmessage "stack"
func (sc *StaticConfig) checkSubscribed(xAddr xco.Address, topic string) bool {

	if len(sc.xmppMessageStack[xAddr]) == 0 {
		return false
	}

	for _, value := range sc.xmppMessageStack[xAddr] {
		if strings.Compare(topic, value) == 0 {
			return true
		}
	}
	return false
}

//answer back all xmpp petitions subscribed to topic
func (sc *StaticConfig) getAddresses(topic string) []xco.Address {

	addresses := []xco.Address{}

	for i, value := range sc.xmppMessageStack {

		for _, v := range value {
			if v == topic {
				addresses = append(addresses, i)
			}
		}

	}
	return addresses
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
