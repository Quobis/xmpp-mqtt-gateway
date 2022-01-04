package main

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	//"errors"
	//mqtt "github.com/eclipse/paho.mqtt.golang"
	xco "github.com/sheenobu/go-xco"
	"gosrc.io/xmpp"
)

var mockConfig StaticConfig

// func TestProcessStanza(t *testing.T) {

// 	mockConfig.mqttMessageStack = make(map[string]string)
// 	mockConfig.xmppMessageStack = make(map[xco.Address][]string)

// 	mockConfig.config.Mqtt = MQTTConfig{
// 		Broker:   "localhost",
// 		Port:     1883,
// 		ClientID: "Client",
// 		Username: "guest",
// 		Password: "guest",
// 	}

// 	opts := mqtt.NewClientOptions()
// 	opts.AddBroker(mockConfig.config.Mqtt.Broker + ":" + fmt.Sprintf("%d", mockConfig.config.Mqtt.Port))
// 	opts.SetClientID(mockConfig.config.Mqtt.ClientID)

// 	c := mqtt.NewClient(opts)

// 	if token := c.Connect(); token.Wait() && token.Error() != nil {
// 		t.Fatalf("Error on Client.Connect(): %v", token.Error())
// 	}

// 	// Disconnect should return within 250ms and calling a second time should not block
// 	disconnectC := make(chan struct{}, 1)
// 	go func() {
// 		c.Disconnect(250)
// 		c.Disconnect(5)
// 		close(disconnectC)
// 	}()

// 	select {
// 	case <-time.After(time.Millisecond * 300):
// 		t.Errorf("disconnect did not finnish within 300ms")
// 	case <-disconnectC:
// 	}

// 	msg := &xco.Message{
// 		XMLName: xml.Name{
// 			Local: "message",
// 			Space: "jabber:component:accept",
// 		},

// 		Header: xco.Header{
// 			From: &xco.Address{
// 				LocalPart:  "guest",
// 				DomainPart: "guest",
// 			},
// 			To: &xco.Address{
// 				LocalPart:  "localhost",
// 				DomainPart: "mqtt.gateway.com",
// 			},
// 			ID: NewId(),
// 		},
// 		Type: "chat",
// 		Body: "GET devices",
// 	}

// 	err := mockConfig.processStanza(msg)
// 	ok(t, err)

// }

func TestProcessStanza(t *testing.T) {

	type args struct {
		topic   string
		message string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Case devices",
			args: args{
				topic:   "GET devices",
				message: "[\"device_1\", \"device_2\"]",
			},
			want: "smartgrid/listOfDevices",
		},
		{
			name: "Case values",
			args: args{
				topic:   "GET values bat_01",
				message: "{\"device_id\":\"bat_01\", \"power\": 1111, \"type\":\"Battery\"}",
			},
			want: "smartgrid/bat_01",
		},
		{
			name: "Case variable",
			args: args{
				topic:   "GET power from bat_01",
				message: "{\"power\": 1111, \"type\":\"Battery\"}",
			},
			want: "smartgrid/bat_01/power",
		},
	}
	mockConfig.mqttMessageStack = make(map[string]string)
	mockConfig.xmppMessageStack = make(map[xco.Address][]string)

	mockStanza := &xco.Message{
		XMLName: xml.Name{
			Local: "message",
			Space: "jabber:component:accept",
		},

		Header: xco.Header{
			From: &xco.Address{
				LocalPart:  "ivan",
				DomainPart: "chat.gateway.com",
			},
			To: &xco.Address{
				LocalPart:  "",
				DomainPart: "mqtt.gateway.com",
			},
			ID: NewId(),
		},
		Type: "chat",
		Body: "",
	}

	mockConfig.mqttReadyCh = make(chan bool)
	mockConfig.xmppReadyCh = make(chan bool)

	//sigs := make(chan os.Signal, 1)
	//done := make(chan bool, 1)

	//signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	mockConfig.config.Mqtt = MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		ClientID: "client",
		Username: "guest",
		Password: "guest",
	}

	mockConfig.config.Xmpp = ConfigXmpp{
		Host:   "localhost",
		Name:   "mqtt.gateway.com",
		Port:   1111,
		Secret: "secret",
	}

	//init mock servers
	h := func(t *testing.T, sc *xmpp.ServerConn) {}
	testComponentAddress := fmt.Sprintf("%v:%v", mockConfig.config.Xmpp.Host, mockConfig.config.Xmpp.Port)
	mock := xmpp.ServerMock{}
	mock.Start(t, testComponentAddress, h)

	mockConfig.runXmppProcess()
	<-mockConfig.xmppReadyCh
	fmt.Println("treu")
	mockConfig.runMqttProcess()
	<-mockConfig.mqttReadyCh

	for i, tt := range tests {

		mockStanza.Body = tt.args.topic

		//checking results on driven tables
		t.Run(strconv.Itoa(i)+": "+tt.name, func(t *testing.T) {
			_, err := mockConfig.processStanza(mockStanza)

			for i, value := range mockConfig.xmppMessageStack[*mockStanza.From] {
				if strings.Compare(value, tt.want) == 0 {
					break
				} else if i == len(mockConfig.xmppMessageStack[*mockStanza.From])-1 {
					t.Errorf("Topic obtained = %v, topic wanted %v", value, tt.want)
				}

			}
			ok(t, err)
		})
	}

	//Testing after the subscription, giving some example test

	for i, tt := range tests {

		mockStanza.Body = tt.args.topic

		fmt.Println()

		mockConfig.mqttMessageStack[tt.want] = tt.args.message

		t.Run(strconv.Itoa(i)+": "+tt.name, func(t *testing.T) {
			testStanza, err := mockConfig.processStanza(mockStanza)

			if strings.Compare(testStanza.Body, tt.args.message) != 0 {

				t.Errorf("Message obtained = %v, message wanted %v", testStanza.Body, tt.args.message)
			}
			ok(t, err)
		})

	}

}

// func equals(tb testing.TB, exp, act interface{}) {
// 	if !reflect.DeepEqual(exp, act) {
// 		_, file, line, _ := runtime.Caller(1)
// 		fmt.Printf("\n\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
// 		tb.FailNow()
// 	}
// }

func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("%s:%d: unexpected error: %s\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}
