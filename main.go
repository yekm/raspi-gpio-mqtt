package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	flag "github.com/spf13/pflag"
)

var port_settings map[int]PortSettings

func main() {
	fhelp := flag.BoolP("help", "h", false, "show help")
	fports := flag.StringSliceP("ports", "p", []string{}, "port numbers with component names and names, see example below")
	fprefix := flag.StringP("discoveryprefix", "d", "homeassistant", "home assistant discovery prefix")
	fbroker := flag.StringP("mqtt", "m", "tcp://localhost:1883", "mqtt broker")
	fnode := flag.StringP("name", "n", "raspi-gpio", "mqtt node name")
	fqos := flag.IntP("qos", "q", 1, "mqtt QoS")
	flag.Parse()

	if *fhelp == true {
		flag.Usage()
		fmt.Print("\nport component names synonyms:\n",
			"\tSensor: i, in, binary_sensor\n",
			"\tSwitch: o, out, switch\n",
			"Example: -p 12:o:lights,18:i:some_button\n")
		fmt.Println("gpio handling is done via sysfs, see github.com/brian-armstrong/gpio")
		os.Exit(0)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ss := SystemSettings{
		DiscoveryPrefix: *fprefix,
		NodeID:          *fnode,
		Qos:             byte(*fqos),
	}

	mqtt := MqttClient{Broker: *fbroker}
	mqtt.connect("sysfs-gpio")

	port_settings = make(map[int]PortSettings)

	mevent := make(chan MqttEvent)

	var buttons []uint = make([]uint, 0)
	for _, v := range *fports {
		s := strings.Split(v, ":")
		pin, err := strconv.Atoi(s[0])
		if err != nil {
			log.Fatal("wring pin number ", s[0])
		}
		component, name := s[1], s[2]
		switch component {
		case "i", "in", "binary_sensor":
			port_settings[pin] = ss.GetPort(name, uint(pin), PinBinarySensor)
			buttons = append(buttons, uint(pin))
			break
		case "o", "out", "switch":
			port_settings[pin] = ss.GetPort(name, uint(pin), PinSwitch)
			break
		default:
			log.Fatal("wrong port component name ", component)
			break
		}
	}

	for _, v := range port_settings {
		mqtt.AdvertisePort(v, mevent)
	}

	button_ch := debounce(watchPins(buttons))

	fmt.Println("Ready...")
	for {
		select {
		case e := <-button_ch:
			fmt.Println(fmt.Sprintf("button event pin:%d val:%d", e.Pin, e.Value))
			ps := port_settings[int(e.Pin)]
			ps.Value = e.Value
			mqtt.PubState(ps)
			break
		case sig := <-sigs:
			fmt.Println(sig)
			for _, v := range port_settings {
				mqtt.UnAdvertisePort(v)
				v.Pin.Cleanup()
			}
			os.Exit(0)
			break
		case me := <-mevent:
			fmt.Println("got mevent ", me.Topic, me.Payload)
			t := strings.Split(me.Topic, "/") // ugh
			//oid := strings.Split(t[3], "_")
			pin, err := strconv.Atoi(t[3])
			if err != nil {
				log.Fatal("wrong pin n ", t[3])
			}
			ps := port_settings[pin]
			if ps.Type != PinSwitch {
				log.Fatal("cmd to sensor?")
			}
			switch me.Payload {
			case "ON":
				fmt.Printf("setting pin %d high\n", pin)
				ps.Pin.High()
				ps.Value = 1
				break
			case "OFF":
				fmt.Printf("setting pin %d low\n", pin)
				ps.Pin.Low()
				ps.Value = 0
				break
			default:
				log.Fatal("bad payload ", me.Payload)
			}
			mqtt.PubState(ps)
			break
		}
	}
}
