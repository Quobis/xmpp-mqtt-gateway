# xmpp-mqtt-gateway

This gateway receives commands via XMPP and translates them into MQTT subscriptions to specific topics of IoT devices. It leverages Golang concureency features (namely goroutines and channels) to handle the messages of different protocols.

On top of this gateway general process we applied these specific rules for our use case. However these rules can be easily extended or changed to be able to cover other use cases or even inter-work different procotols. 

## Basic architecture
This gateway consists of a main routine which receives the messages from channels where parallel MQTT and XMPP routines publish the messages coming from the server. Then specific handlers process the messages and generates the outgoing messages. The gateway logic is included in those handlers. Following the best practices, and as recommended by the library authors, all the message processing is done in goroutines. 

## Dependencies
- MQTT library: https://github.com/eclipse/paho.mqtt.golang
- XMPP library: https://github.com/sheenobu/go-xco
## Aknowledgement

This project has been funded by ITEA3 Scratch project.

