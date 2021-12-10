package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

func (c *mqttClient) mqttSub(topic string) error {
	if len(topic) == 0 {
		fmt.Printf("Not subscribing to empty topic! ")
	}
	fmt.Printf("Subscribing to topic %s\n", topic)

	token := c.Client.Subscribe(topic, 1, nil)

	// go func() {
	// 	<-token.Done()
	// 	//blocking the goroutine
	// 	if token.Error() != nil {
	// 		log.Print(token.Error())
	// 	}
	// }()

	//token.Wait()
	fmt.Printf("\nClient %s, \nsubscribed to topic %s", c.Username, topic)

	return token.Error()
}

func (c *mqttClient) mqttPublish(message, topic string) error {

	fmt.Sprintf("\nPublishing to topic: %s", topic)

	token := c.Client.Publish(topic, 0, false, message)
	//waits indefinetly until the message is sent to the broker and ack back to the client

	token.Wait()
	fmt.Sprintf("Published %s,\nwith topic: %s", message, topic)

	return token.Error()
}

func (c *mqttClient) runMqttClient(sc *StaticConfig) <-chan struct{} {
	opts := mqtt.NewClientOptions()

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	//mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	//check values!!!!
	opts.AddBroker(fmt.Sprintf(c.Url))
	opts.SetClientID(c.ClientID)
	opts.SetUsername(c.Username)
	opts.SetPassword(c.Password)

	c.messagePubHandler = func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Received message: %s \nfrom topic: %s\n", msg.Payload(), msg.Topic())
		go func() { c.gatewayRx <- &msg }()
		//ANTON: prepare XMPP message including this information back
	}

	opts.SetDefaultPublishHandler(c.messagePubHandler)

	c.connectHandler = func(client mqtt.Client) {
		fmt.Println("Connected to MQTT broker")
	}

	opts.OnConnect = c.connectHandler

	c.connectLostHandler = func(client mqtt.Client, err error) {
		fmt.Printf("Connection against MQTT broker lost due to error: %v", err)
	}

	opts.OnConnectionLost = c.connectLostHandler

	c.Client = mqtt.NewClient(opts)

	sc.mqttClient = *c

	//Subscribe to Scratch project topic - To Be improved
	//c.mqttSub("scratch")
	fmt.Println("Inside runMqttClient: " + c.Username)

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

		// c.Client.Subscribe("/smartgrid/listOfDevices", 1, nil)
		// c.Client.Subscribe("/smartgrid/device1/variable1", 1, nil)
		// c.Client.Subscribe("/smartgrid/device1", 1, nil)

		//with this we keep the client connected until a SIGTERM
		//signal is received
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		c.Client.Disconnect(250)

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
