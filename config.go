package main

type Config struct {
	Xmpp        ConfigXmpp         `toml:"xmpp"`
	Mqtt        MQTTConfig         `toml:"mqtt"`
	SippoServer *SippoServerConfig `toml:"sipposerver"`
}

type ConfigXmpp struct {
	Host   string `toml:"host"`
	Name   string `toml:"name"`
	Port   int    `toml:"port"`
	Secret string `toml:"secret"`
}

type MQTTConfig struct {
	Broker   string `toml:"broker"`
	Port     int    `toml:"port"`
	ClientID string `toml:"clientID"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type SippoServerConfig struct {
	Host     string `toml:"host"`
	User     string `toml:"user"`
	Password string `toml:"password"`
}
