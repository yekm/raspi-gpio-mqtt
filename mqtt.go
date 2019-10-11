package main

import (
	"fmt"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MqttClient is a wrapper for the eclipse mqtt library
type MqttClient struct {
	Broker string
	client mqtt.Client
}

type MqttEvent struct {
	Topic, Payload string
}

func (c *MqttClient) connect(clientID string) bool {

	hostname, _ := os.Hostname()
	client_id := fmt.Sprintf("%s-%s", hostname, clientID)
	fmt.Printf("Connecting to %s as %s\n", c.Broker, client_id)

	mqtt.ERROR = log.New(os.Stderr, "mqtt:", 0)
	opts := mqtt.NewClientOptions().AddBroker(c.Broker)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetClientID(client_id)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(2 * time.Minute)
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("MQTT default publish handler: [%s] -> [%s]", msg.Topic(), string(msg.Payload()))
	})
	c.client = mqtt.NewClient(opts)
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		//fmt.Errorf(token.Error())
		return false // error.New(token.Error())
	}
	return true
}

func (c *MqttClient) publish(topic string, message interface{}) {
	c.client.Publish(topic, 0, false, message)
}

func (c *MqttClient) AdvertisePort(port PortSettings, evch chan MqttEvent) {
	switch port.Type {
	case PinBinarySensor:
		break
	case PinSwitch:
		c.client.Subscribe(port.CommandTopic(), 0, func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("CMD TOPIC [%s] %s\n", msg.Topic(), string(msg.Payload()))
			evch <- MqttEvent{msg.Topic(), string(msg.Payload())}
		})
		break
	}
	c.client.Publish(port.ConfigTopic(), port.ss.Qos, false, port.ConfigPayload())
}

func (c *MqttClient) UnAdvertisePort(port PortSettings) {
	c.client.Publish(port.ConfigTopic(), port.ss.Qos, false, "")
}

func (c *MqttClient) PubState(port PortSettings) {
	c.client.Publish(port.StateTopic(), port.ss.Qos, false, port.StatePayload())
}
