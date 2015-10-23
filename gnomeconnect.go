package main

import (
	"log"
	"github.com/emersion/go-kdeconnect/engine"
	"github.com/emersion/go-kdeconnect/plugin"
	"github.com/emersion/go-kdeconnect/network"
	"github.com/godbus/dbus"
	"github.com/esiqveland/notify"
)

func getDeviceIcon(device *network.Device) string {
	switch device.Type {
	case "phone":
		return "phone"
	case "tablet":
		return "pda" // TODO: find something better
	case "desktop":
		return "computer"
	default:
		return ""
	}
}

func newNotification() notify.Notification {
	return notify.Notification{
		AppName: "GNOMEConnect",
	}
}

func main() {
	battery := plugin.NewBattery()
	ping := plugin.NewPing()

	go (func() {
		conn, err := dbus.SessionBus()
		if err != nil {
			panic(err)
		}

		notifier := notify.New(conn)

		for {
			select {
			case event := <-ping.Incoming:
				log.Println("New ping from device:", event.Device.Name)

				n := newNotification()
				n.AppIcon = getDeviceIcon(event.Device)
				n.Summary = "Ping from "+event.Device.Name
				notifier.SendNotification(n)
			case event := <-battery.Incoming:
				log.Println("Battery:", event.Device.Name, event.BatteryBody)

				if event.ThresholdEvent == plugin.BatteryThresholdEventLow {
					n := newNotification()
					n.AppIcon = "battery-caution"
					n.Summary = event.Device.Name+" has low battery"
					notifier.SendNotification(n)
				}
			}
		}
	})()

	hdlr := plugin.NewHandler()
	hdlr.Register(battery)
	hdlr.Register(ping)

	e := engine.New(hdlr)
	e.Listen()
}
