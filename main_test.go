package main

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"

	//"errors"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	xco "github.com/sheenobu/go-xco"
	"gosrc.io/xmpp"
)

type args struct {
	topic   string
	message string
}

var mockConfig StaticConfig

func TestMqttToStanza(t *testing.T) {

	mockConfig.rxMqttCh = make(chan *mqtt.Message)
	mockConfig.rxXmppCh = make(chan *xco.Message)

	tests := []struct {
		name string
		chat string
		args args
		want string
	}{
		{
			name: "Case devices",
			args: args{
				topic:   "smartgrid/listOfDevices",
				message: "[\"device_1\", \"device_2\"]",
			},

			want: "\nDevices: \n\tdevice_1\n\tdevice_2\n",
		},
		{
			name: "Case values",
			args: args{
				topic:   "smartgrid/bat_01",
				message: "{\"device_id\":\"bat_01\", \"power\": 1111, \"type\":\"Battery\"}",
			},
			//this method fails "randomly" probably because of unmarhall (changes the order of the fields)
			want: "\nbat_01:\n\tdevice_id : bat_01\n\tpower : 1111\n\ttype : Battery\n",
		},
		{
			name: "Case variable",
			args: args{
				topic:   "smartgrid/bat_01/power",
				message: "{\"power\": 1111, \"type\":\"Battery\"}",
			},
			want: "\nBattery: bat_01\n\tpower: 1111",
		},
	}

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

	err := mockConfig.serversUp(t)
	ok(t, err)

	fmt.Println()
	//client subscribing directly to the "table" topics
	t.Run("Subscribing test.", func(t *testing.T) {

		for _, tt := range tests {
			token := mockConfig.mqttClient.Client.Subscribe(tt.args.topic, 1, nil)
			err = token.Error()
			mockConfig.xmppMessageStack[*mockStanza.From] = append(mockConfig.xmppMessageStack[*mockStanza.From], tt.args.topic)
			mockConfig.mqttMessageStack[tt.args.topic] = tt.args.message
			ok(t, err)
			t.Logf("\nClient %s subscribed to topic: %s", mockConfig.mqttClient.Username, tt.args.topic)
		}
	})

	//client publishing directly on the "table" topics
	for _, tt := range tests {
		fmt.Println()
		t.Run(tt.name, func(t *testing.T) {

			token := mockConfig.mqttClient.Client.Publish(tt.args.topic, 1, false, tt.args.message)
			err = token.Error()
			ok(t, err)

			//t.Logf("\nPublished %s on topic: %s", tt.args.message, tt.args.topic)
			select {
			case message := <-mockConfig.rxMqttCh:
				fmt.Println("MQTT message received.")

				testStanza, err := mockConfig.mqttToStanza(message)
				ok(t, err)

				if strings.Compare(testStanza.Body, tt.want) != 0 {

					t.Errorf("Message obtained = %v, message wanted %v", testStanza.Body, tt.want)
				}

			case <-time.After(3 * time.Second):
				t.Fatal("Timeout waitig for mqtt message on: " + tt.name)

				ok(t, err)
			}
		})
	}

	fmt.Println()

}

func TestProcessStanza(t *testing.T) {

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

	err := mockConfig.serversUp(t)
	ok(t, err)

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

func (sc *StaticConfig) serversUp(t *testing.T) error {

	sc.mqttMessageStack = make(map[string]string)
	sc.xmppMessageStack = make(map[xco.Address][]string)

	sc.mqttReadyCh = make(chan bool)
	sc.xmppReadyCh = make(chan bool)

	sc.config.Mqtt = MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		ClientID: "Client",
		Username: "guest",
		Password: "guest",
	}

	sc.config.Xmpp = ConfigXmpp{
		Host:   "localhost",
		Name:   "mqtt.gateway.com",
		Port:   1111,
		Secret: "secret",
	}

	//init mock servers
	h := func(t *testing.T, sc *xmpp.ServerConn) {}
	testComponentAddress := fmt.Sprintf("%v:%v", sc.config.Xmpp.Host, sc.config.Xmpp.Port)
	mock := xmpp.ServerMock{}
	mock.Start(t, testComponentAddress, h)

	wg := sync.WaitGroup{}
	waitCh := make(chan struct{})
	wg.Add(2)

	go func() {
		go func() {
			defer wg.Done()
			sc.runXmppProcess()
			<-sc.xmppReadyCh
		}()
		go func() {
			defer wg.Done()
			sc.runMqttProcess()
			<-sc.mqttReadyCh
		}()
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		fmt.Println("XMPP and MQTT ready")
	case <-time.After(5 * time.Second):
		return errors.New("Connection exceeded timeout. Aborting")
	}

	return nil
}
