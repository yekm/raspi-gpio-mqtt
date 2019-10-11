package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/brian-armstrong/gpio"
)

// <discovery_prefix>/<component>/[<node_id>/]<object_id>/config

// PinType hw type: binary_sensor (gpio in) / switch (gpio out) / etc (pwm i2c)
type PinType int

const (
	PinBinarySensor PinType = iota
	PinSwitch
)

// SystemSettings global settings, embedded in PortSettings
type SystemSettings struct {
	DiscoveryPrefix string
	NodeID          string
	Qos             byte
}

type HassPayload struct {
	Name string `json:"name"`
	//Device_class  string `json:"device_class,omitepmty"`
	State_topic   string `json:"state_topic,omitempty"`
	Command_topic string `json:"command_topic,omitempty"`
}

// PortSettings type and state of the port will be used in event exchange
type PortSettings struct {
	Type     PinType
	Port     uint
	Value    uint
	Name     string
	Pin      gpio.Pin
	ConfigPl *HassPayload
	ss       *SystemSettings
}

type HassSettings struct {
	Component, ObjectID string
}

type PortHassSettings map[PinType]HassSettings

func PHS() PortHassSettings {
	return PortHassSettings{
		PinBinarySensor: HassSettings{
			Component: "binary_sensor",
			//ObjectID:  "bs_",
		},
		PinSwitch: HassSettings{
			Component: "switch",
			//ObjectID:  "sw_",
		},
	}
}

// GetPort create PortSettings strcut from pin number and hw type
func (ss *SystemSettings) GetPort(name string, port uint, ptype PinType) PortSettings {
	ps := PortSettings{
		Type: ptype,
		Port: port,
		Name: name,
		ss:   ss,
	}

	ps.ConfigPl = &HassPayload{
		Name:        name,
		State_topic: ps.StateTopic(),
		//Device_class:  "power", // TODO move to args https://www.home-assistant.io/integrations/binary_sensor/#device-class
	}

	switch ptype {
	case PinBinarySensor:
		//ps.Pin = gpio.NewInput(port)
		break
	case PinSwitch:
		ps.Pin = gpio.NewOutput(port, false)
		ps.ConfigPl.Command_topic = ps.CommandTopic()
		break
	default:
		log.Fatalf("wrong pin type %d", ptype)
		break
	}

	return ps
}

func (ps *PortSettings) NodeTopic() string {
	var sb strings.Builder
	sb.WriteString(ps.ss.DiscoveryPrefix)
	sb.WriteString("/")
	sb.WriteString(PHS()[ps.Type].Component)
	sb.WriteString("/")
	sb.WriteString(ps.ss.NodeID)
	sb.WriteString("/")
	//sb.WriteString(PHS()[ps.Type].ObjectID)
	sb.WriteString(strconv.Itoa(int(ps.Port)))
	return sb.String()
}

func (ps *PortSettings) ConfigTopic() string {
	return ps.NodeTopic() + "/config"
}
func (ps *PortSettings) CommandTopic() string {
	return ps.NodeTopic() + "/set"
}
func (ps *PortSettings) StateTopic() string {
	return ps.NodeTopic() + "/state"
}

func (ps *PortSettings) ConfigPayload() string {
	j, _ := json.Marshal(ps.ConfigPl)
	return string(j)
}

func (ps *PortSettings) StatePayload() string {
	if ps.Value == 0 {
		return "OFF"
	}
	return "ON"
}
