# xmpp-mqtt-gateway

This gateway receives commands via XMPP and translates them into MQTT subscriptions to specific topics of IoT devices. It leverages Golang concurrency features (namely goroutines and channels) to handle the messages of different protocols.

On top of this gateway general process we applied these specific rules for our use case. However these rules can be easily extended or changed to be able to cover other use cases or even inter-work different protocols.

## Basic architecture
This gateway consists of a main routine which receives the messages from channels where parallel MQTT and XMPP routines publish the messages coming from the server. Then specific handlers process the messages and generate the outgoing messages. The gateway logic is included in those handlers. Following the best practices, and as recommended by the library authors, all the message processing is done in goroutines.
## Compile and execution process
1. Download the repository from Github.
2. Execute go build
3. You will get a xmpp-mqtt-gateway executable file.
4. The service will be deployed as a container, but this is still a WIP.

In order to carry out integration tests, both Rabbitmq MQTT server and Prosody XMPP server must be launched: 
docker-compose up -d 
The service will be deployed as a container, but this is still a WIP.
Next step is cloning NVISOsecurity’s MQTT         proxy, that is IOXY, this tool will run a MQTT server acting as a Man In The Middle.
git clone https://github.com/NVISO-BE/IOXY
cd IOXY/ioxy && go build .






In order to encapsulate the MQTT traffic with TLS we are using self-signed certificates located on xmpp-mqtt-gateway/certs. The usage of tls-gen tool is highly recommended for this purpose.
When it is done, ioxy is launched using this command, including the location of the tls certs and keys that are needed:
sudo ./ioxy                               mqtts                                   -mqtts-port             8885                -mqtts-cert             ../../certs/server_certificate.pem          -mqtts-key             ../../certs/server_key.pem           -mqtts-ca               ../../certs/ca_certificate.pem             broker                                  -mqtt-broker-tls                        -mqtt-broker-host   localhost             -mqtt-broker-port   8883                -mqtt-broker-cert   ../../certs/client_certificate.pem            -mqtt-broker-key        ../../certs/client_key.pem  gui
Now IOXY can be administered using a gui on localhost:1111 by default.
In order to test the gateway we use MQTTX as an MQTT broker and Psi as a chat provider. The psi appimage is on the repository’s root directory.
Only mqtt is encapsulated using tls at the moment so the connection between IOXY and the mqtt client should be configured that way.

## Dependencies
* MQTT library: https://github.com/eclipse/paho.mqtt.golang
* XMPP library: https://github.com/sheenobu/go-xco 
* IOXY:  https://github.com/NVISOsecurity/IOXY
* TLS-GEN: https://github.com/michaelklishin/tls-gen


## Acknowledgement
This project has been developed within the ITEA3 Scratch project.
## Debug
MQTT debugging is available if by uncommenting the mqtt.DEBUG= log.New(os.Stdout, "[DEBUG] ", 0) line on mqtt.go