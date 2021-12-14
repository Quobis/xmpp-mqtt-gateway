# xmpp-mqtt-gateway

This gateway receives commands via XMPP and translates them into MQTT subscriptions to specific topics of IoT devices. It leverages Golang concurrency features (namely goroutines and channels) to handle the messages of different protocols.

On top of this gateway general process we applied these specific rules for our use case. However these rules can be easily extended or changed to be able to cover other use cases or even inter-work different procotols. 

## Basic architecture
This gateway consists of a main routine which receives the messages from channels where parallel MQTT and XMPP routines publish the messages coming from the server. Then specific handlers process the messages and generates the outgoing messages. The gateway logic is included in those handlers. Following the best practices, and as recommended by the library authors, all the message processing is done in goroutines. 

## Compile and execution process 
1. Download the repository from Github.
2. Execute `go build`
3. You will get a `xmpp-mqtt-gateway` executable file.
4. The service will be deployed as a container, but this is still a WIP. 

In order to carry out integration tests, both Mosquito MQTT server and Prosody XMPP server must be launched:
`docker run -it -p 1883:1883 -p 9001:9001 -v /home/antonroman/src/xmpp-mqtt-gateway/mosquitto-conf/mosquitto.conf:/mosquitto/config/mosquitto.conf  eclipse-mosquitto`
`docker run --name prosody -p 5222:5222 -p 5347:5347 -v /home/antonroman/src/xmpp-mqtt-gateway/prosody-conf/prosody.cfg.lua:/etc/prosody/prosody.cfg.lua:ro -v /home/antonroman/src/xmpp-mqtt-gateway/prosody-conf/certs:/etc/prosody/certs:ro prosody-anton`

The service will be deployed as a container, but this is still a WIP. 

## Dependencies
- MQTT library: https://github.com/eclipse/paho.mqtt.golang
- XMPP library: https://github.com/sheenobu/go-xco

## Aknowledgement
This project has been developed within the ITEA3 Scratch project.

## Debug
MQTT debugging is available if by uncommenting the `mqtt.DEBUG= log.New(os.Stdout, "[DEBUG] ", 0)` line on mqtt.go