package main

import (
	"fmt"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// type mqttReceived struct {
// 	client mqtt.Client
// 	msg    mqtt.Message
// }

type mqttClient struct {
	Url                string
	Password           string
	Username           string
	ClientID           string
	Client             mqtt.Client
	messagePubHandler  mqtt.MessageHandler
	connectHandler     mqtt.OnConnectHandler
	connectLostHandler mqtt.ConnectionLostHandler
	gatewayRx          chan<- *mqtt.Message
}

// func (c *mqttClient) mqttSub(topic string) {
// 	//topic := "topic/test"
// 	if len(topic) == 0 {
// 		fmt.Printf("Not subscribing to empty topic! ")
// 	}
// 	fmt.Printf("Subscribing to topic %s", topic)
// 	fmt.Println(c, "\t", pa, "\n")

// 	token := c.Client.Subscribe(topic, 1, nil)
// 	//waiting in gotoutine to minimize blocking
// 	go func() {
// 		<-token.Done()
// 		if token.Error() != nil {
// 			log.Print(token.Error()) // Use your preferred logging technique (or just fmt.Printf)
// 		}
// 	}()

// 	fmt.Printf("Subscribed to topic %s", topic)
// }

// func (c *mqttClient) messagePubHandler mqtt.messagePubHandler =  (c *mqttClient) func (client mqtt.Client, msg mqtt.Message) {
// 	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
// 	//ANTON: prepare XMPP message including this information back
// } */

// func (c *mqttClient) connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
// 	fmt.Println("Connected to MQTT broker")
// }

// func (c *mqttClient) connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
// 	fmt.Printf("Connection against MQTT broker lost: %v", err)
// }

func (c *mqttClient) runMqttClient(sc *StaticConfig) <-chan struct{} {
	opts := mqtt.NewClientOptions()

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	//check values!!!!
	opts.AddBroker(fmt.Sprintf(c.Url))
	opts.SetClientID(c.ClientID)
	opts.SetUsername(c.Username)
	opts.SetPassword(c.Password)

	c.messagePubHandler = func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		go func() { c.gatewayRx <- &msg }()
		//ANTON: prepare XMPP message including this information back
	}

	c.connectHandler = func(client mqtt.Client) {
		fmt.Println("Connected to MQTT broker")
	}

	c.connectLostHandler = func(client mqtt.Client, err error) {
		fmt.Printf("Connection against MQTT broker lost due to error: %v", err)
	}

	//Subscribe to Scratch project topic - To Be improved
	//c.mqttSub("scratch")
	fmt.Println("Inside runMqttClient")

	healthCh := make(chan struct{})
	go func() {
		defer func() {
			recover()
			close(healthCh)
		}()
		fmt.Println("Connecting to MQTT broker")
		if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
		fmt.Println("Connected to MQTT broker")

	}()
	return healthCh
}

// func (c *mqttClient) onReceivedMessage(x *xco.Component, m *xco.Message) error {
// 	log.Printf("Message: %+v, To: %s", m, m.To.LocalPart)
// 	if m.Body == "" {
// 		log.Printf("  ignoring message with empty body")
// 		return nil
// 	}
// 	go func() { c.gatewayRx <- m }()
// 	return nil
// }
