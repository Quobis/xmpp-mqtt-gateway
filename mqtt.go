package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

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
	fmt.Printf("Subscribing to topic %s \n", topic)

	token := c.Client.Subscribe(topic, 1, nil)

	// go func() {
	// 	<-token.Done()
	// 	if token.Error() != nil {
	// 		log.Print(token.Error())
	// 	}
	// }()

	//token.Wait()
	if token.Error() == nil {

		fmt.Printf("Client %s, subscribed to topic %s\n", c.Username, topic)
	}

	return token.Error()
}

// func (c *mqttClient) mqttPublish(message, topic string) error {

// 	fmt.Printf("\nPublishing to topic: %s", topic)

// 	token := c.Client.Publish(topic, 0, false, message)
// 	//waits indefinetly until the message is sent to the broker and ack back to the client

// 	token.Wait()
// 	fmt.Printf("Published %s,\nwith topic: %s", message, topic)

// 	return token.Error()
// }

func (c *mqttClient) runMqttClient(sc *StaticConfig) <-chan struct{} {
	opts := mqtt.NewClientOptions()

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	//mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	//check values!!!!
	opts.AddBroker("ssl://" + fmt.Sprintf(c.Url))
	opts.SetUsername(c.Username)
	opts.SetPassword(c.Password)

	c.messagePubHandler = func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Received message: %s \nfrom topic: %s\n", msg.Payload(), msg.Topic())
		go func() {
			c.gatewayRx <- &msg
		}()
		//ANTON: prepare XMPP message including this information back
	}

	opts.SetDefaultPublishHandler(c.messagePubHandler)

	c.connectHandler = func(client mqtt.Client) {
		fmt.Println("Connected to MQTT broker")
		sc.mqttReadyCh <- true
	}

	//tls configuration
	tlsConfig := NewTlsConfig()
	opts.SetClientID(c.ClientID).SetTLSConfig(tlsConfig)

	opts.OnConnect = c.connectHandler

	timeout, _ := time.ParseDuration("10s")

	opts.SetConnectTimeout(timeout)

	opts.SetOrderMatters(false)

	opts.AutoReconnect = false

	c.connectLostHandler = func(client mqtt.Client, err error) {
		fmt.Printf("Connection against MQTT broker lost due to error: %v\n", err)
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

		//with this we keep the client connected until a SIGTERM
		//signal is received

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		c.Client.Disconnect(250)

	}()
	return healthCh
}

func NewMessageID() uint16 {
	rand.Seed(time.Now().UnixNano())
	return uint16(rand.Intn(65536))
}

func NewTlsConfig() *tls.Config {
	certpool := x509.NewCertPool()
	ca, err := ioutil.ReadFile("./certs/ca_certificate.pem")
	if err != nil {
		certpool.AppendCertsFromPEM(ca)
	}
	// Import client certificate/key pair
	cert, err := tls.LoadX509KeyPair("./certs/client_certificate.pem", "./certs/client_key.pem")
	if err != nil {
		panic(err)
	}
	// // Just to print out the client certificate..
	// cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(cert.Leaf)

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}
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
