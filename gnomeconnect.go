package main

import (
	"log"
	"os"
	"io/ioutil"
	"crypto/rsa"
	"github.com/emersion/go-kdeconnect/crypto"
	"github.com/emersion/go-kdeconnect/engine"
	"github.com/emersion/go-kdeconnect/plugin"
	"github.com/emersion/go-kdeconnect/network"
	"github.com/godbus/dbus"
	"github.com/esiqveland/notify"
)

func getPrivateKey() (priv *rsa.PrivateKey, err error) {
	configHomeDir := os.Getenv("XDG_CONFIG_HOME")
	if configHomeDir == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return
		}
		configHomeDir = homeDir+"/.config"
	}

	configDir := configHomeDir+"/gnomeconnect"
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return
	}

	privateKeyFile := configDir+"/private.pem"
	raw, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		priv, err = crypto.GenerateKey()
		if err != nil {
			return
		}

		raw, err = crypto.MarshalPrivateKey(priv)
		if err != nil {
			return
		}

		err = ioutil.WriteFile(privateKeyFile, raw, 0644)
		return
	}

	priv, err = crypto.UnmarshalPrivateKey(raw)
	return
}

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
	config := engine.DefaultConfig()

 	priv, err := getPrivateKey()
	if priv == nil {
		log.Fatal("Could not get private key:", err)
	}
	if err != nil {
		log.Println("Warning: error while getting private key:", err)
	}
	config.PrivateKey = priv

	battery := plugin.NewBattery()
	ping := plugin.NewPing()
	notification := plugin.NewNotification()

	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}

	notifier := notify.New(conn)

	go (func() {
		for {
			select {
			case event := <-ping.Incoming:
				log.Println("Ping:", event.Device.Name)

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
			case event := <-notification.Incoming:
				log.Println("Notification:", event.Device.Name, event.NotificationBody)

				if event.IsCancel {
					// TODO: dismiss notification
					break
				}

				n := newNotification()
				n.AppIcon = getDeviceIcon(event.Device)
				n.Summary = "Notification from "+event.AppName+" on "+event.Device.Name
				n.Body = event.Ticker
				notifier.SendNotification(n)

				// TODO: wait for notification dismiss and send message to remote
			}
		}
	})()

	hdlr := plugin.NewHandler()
	hdlr.Register(battery)
	hdlr.Register(ping)
	hdlr.Register(notification)

	e := engine.New(hdlr, config)
	e.Listen()
}
