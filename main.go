package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/sheenobu/go-xco"
	//p "xmpp-gateway/providers"
	//"github.com/BurntSushi/toml"
	//"github.com/pkg/errors"
	//"github.com/sheenobu/go-xco"
)

type StaticConfig struct {
	config      Config
	sippoClient *SippoClient

	//	provider  p.Provider
	xmppComponent Component

	rxMqttCh chan *mqtt.Message
	rxXmppCh chan *xco.Message
}

/* var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
	//ANTON: prepare XMPP message including this information back
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to MQTT broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection against MQTT broker lost: %v", err)
} */

func processStanza(stanza string) {
	if len(stanza) == 0 {
		fmt.Printf("Not processing empty stanza! ")
	}

	// If get values <device>
	// mqttSub() to <device>
	//

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

	//MQTT part
	// var broker = sc.config.Mqtt.Broker
	// var port = sc.config.Mqtt.Port
	// opts := mqtt.NewClientOptions()
	// opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	// opts.SetClientID("go_mqtt_client")
	// opts.SetUsername("emqx")
	// opts.SetPassword("public")
	// opts.SetDefaultPublishHandler(messagePubHandler)
	// opts.OnConnect = connectHandler
	// opts.OnConnectionLost = connectLostHandler
	// client := mqtt.NewClient(opts)

	// if token := client.Connect(); token.Wait() && token.Error() != nil {
	// 	panic(token.Error())
	// }

	// mqttSub(client, "smartgrid/listOfDevices")
	/* 	t := client.Publish("topic", qos, retained, msg)
	   	go func() {
	   		_ = t.Wait() // Can also use '<-t.Done()' in releases > 1.2.0
	   		if t.Error() != nil {
	   			log.Error(t.Error()) // Use your preferred logging technique (or just fmt.Printf)
	   		}
	   	}() */

	//sc.rxHttpCh = make(chan p.RxHttp)
	sc.rxXmppCh = make(chan *xco.Message)
	sc.rxMqttCh = make(chan *mqtt.Message)

	// start goroutines
	gatewayDead := sc.runGatewayProcess()
	xmppDead := sc.runXmppProcess()
	mqttDead := sc.runMqttProcess()
	// httpDead := sc.runHttpProcess()
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

/* func (sc *StaticConfig) runHttpProcess() <-chan struct{} {

	provider, err := sc.getProvider()
	if err != nil {
		msg := fmt.Sprintf("Couldn't choose a provider: %s", err)
		panic(msg)
	}

	return provider.RunHttpServer()
} */

/* func (sc *StaticConfig) getProvider() (p.Provider, error) {
	if sc.config.Vonage != nil {
		vonage := &p.Vonage{
			Endpoint: sc.config.Vonage.Endpoint,
			Token:    sc.config.Vonage.Token,
			Number:   sc.config.Vonage.Number,
			Host:     sc.config.Vonage.Host,
			Port:     sc.config.Vonage.Port,
			RxHttpCh: sc.rxHttpCh,
		}
		sc.provider = vonage
		return vonage, nil
	}
	return nil, errors.New("Need to configure a provider")
} */

func (sc *StaticConfig) runGatewayProcess() <-chan struct{} {
	healthCh := make(chan struct{})

	//go func(rxHttpCh <-chan p.RxHttp, , rxMqttCh <- chan *mqtt.Message) {
	go func(rxXmppCh <-chan *xco.Message, rxMqttCh <-chan *mqtt.Message) {
		defer func() {
			recover()
			close(healthCh)
		}()

		for {

			select {
			case rxXmpp := <-rxXmppCh:
				log.Println("Xmpp stanza received: ", rxXmpp.Body)
				processStanza(rxXmpp.Body)
			case rxMqtt := <-rxMqttCh:
				log.Println("Mqtt message received: ", rxMqtt) //ANTON

			}

			/* 	select {
				case rxHttp := <-rxHttpCh:
			log.Println("Http message")
			errCh := rxHttp.ErrCh()
			switch msg := rxHttp.(type) {
			case *p.VonageInboundRequest:
				log.Println("Inbound Message")
				_, err := sc.whatsapp2Stanza(msg)
				errCh <- err
			case *p.VonageStatusRequest:
				log.Println("Status Message")
				errCh <- nil
			default:
				log.Printf("unexpected type: %#v", rxHttp)
			}
			case rxXmpp := <-rxXmppCh:
				log.Println("Xmpp stanza received: ", rxXmpp.Body)
				//err := sc.stanza2whatsapp(rxXmpp)
				// if err != nil {
				// 	log.Printf("Error: %s", err)
				// }
			} */
			log.Println("gateway looping")
		}
		//}(sc.rxHttpCh, sc.rxXmppCh)
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
	return c.runMqttClient(sc)
}

/* func (sc *StaticConfig) whatsapp2Stanza(whatsapp *p.VonageInboundRequest) (*xco.Message, error) {

	stanzaTo, err := sc.getRecipientAddress(whatsapp.From.Number)

	if err != nil {
		return nil, errors.Wrap(err, "retrieving recipient for message")
	}

	stanzaFrom, err := sc.phone2Address(whatsapp.From.Number)
	if err != nil {
		return nil, errors.Wrap(err, "converting address from phone number")
	}

	stanza := createStanza(stanzaFrom, stanzaTo, whatsapp.Message.Content.Text)

	err = sc.component.xmppComponent.Send(stanza)
	return stanza, errors.Wrap(err, "sending stanza")
}

func (sc *StaticConfig) stanza2whatsapp(stanza *xco.Message) error {
	return sc.provider.Send(stanza.To.LocalPart, "text", stanza.Body)
}

func (sc *StaticConfig) getRecipientAddress(phoneNumber string) (*xco.Address, error) {
	token, err := sc.sippoClient.getAuthToken()

	if err != nil {
		return nil, errors.Wrap(err, "converting address from phone number")
	}
	m, err := sc.sippoClient.GetMeetingsByPhone(token.AccessToken, phoneNumber)

	if err != nil {
		return nil, errors.Wrap(err, "retrieving meetings")
	}
	addr := &xco.Address{
		LocalPart:  m[len(m)-1].User.Username,
		DomainPart: m[len(m)-1].User.Domain,
	}
	return addr, nil
} */

/* func (sc *StaticConfig) phone2Address(phoneNumber string) (*xco.Address, error) {
	addr := &xco.Address{
		LocalPart:  phoneNumber,
		DomainPart: sc.component.Name,
	}
	return addr, nil
} */
