package main

type Battery struct {
	Id_battery string `json:id_battery`
	Power      int    `json:power`
	Capacity   int    `json:capacity`
}

type Solar_kit struct {
	Id_solarkit    string `json:id_solarkit`
	TiltAngle      int    `json:tiltAngle`
	AzimuthAngle   int    `json:aximuthAngle`
	AnnualPv       int    `json:annualPv`
	YearChange     int    `json:yearChange`
	SpectralEffect int    `json:spectralEffect`
	AngleInc       int    `json:angleInc`
	Temperature    int    `json:temperature`
	Losses         int    `json:losses`
}

type Thermostat struct {
	Id_thermostat string `json:id_thermostat`
	CurrentTemp   int    `json:currentTemp`
	Mode          string `json:mode`
	Battery       int    `json:battery`
}
