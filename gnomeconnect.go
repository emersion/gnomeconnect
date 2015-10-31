package main

import (
	"log"
	"os"
	"io/ioutil"
	"encoding/json"
	"github.com/emersion/go-kdeconnect/crypto"
	"github.com/emersion/go-kdeconnect/engine"
	"github.com/emersion/go-kdeconnect/plugin"
	"github.com/emersion/go-kdeconnect/network"
	"github.com/godbus/dbus"
	"github.com/esiqveland/notify"
	"github.com/emersion/go-mpris"
)

func getConfigDir() (configDir string, err error) {
	configHomeDir := os.Getenv("XDG_CONFIG_HOME")
	if configHomeDir == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return
		}
		configHomeDir = homeDir+"/.config"
	}

	configDir = configHomeDir+"/gnomeconnect"
	err = os.MkdirAll(configDir, 0755)
	return
}

func loadPrivateKey() (priv *crypto.PrivateKey, err error) {
	configDir, err := getConfigDir()
	if err != nil {
		return
	}

	priv = &crypto.PrivateKey{}

	privateKeyFile := configDir+"/private.pem"
	raw, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		if err = priv.Generate(); err != nil {
			return
		}

		raw, err = priv.Marshal()
		if err != nil {
			return
		}

		err = ioutil.WriteFile(privateKeyFile, raw, 0644)
		return
	}

	err = priv.Unmarshal(raw)
	return
}

func loadKnownDevices() (knownDevices []*engine.KnownDevice, err error) {
	configDir, err := getConfigDir()
	if err != nil {
		return
	}

	knownDevicesFile, err := os.Open(configDir+"/known-devices.json")
	if err != nil {
		return
	}

	dec := json.NewDecoder(knownDevicesFile)
	err = dec.Decode(&knownDevices)
	return
}

func saveKnownDevices(knownDevices []*engine.KnownDevice) (err error) {
	configDir, err := getConfigDir()
	if err != nil {
		return
	}

	knownDevicesFile, err := os.Create(configDir+"/known-devices.json")
	if err != nil {
		return
	}

	enc := json.NewEncoder(knownDevicesFile)
	err = enc.Encode(knownDevices)
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

 	priv, err := loadPrivateKey()
	if priv == nil {
		log.Fatal("Could not get private key:", err)
	}
	if err != nil {
		log.Println("Warning: error while loading private key:", err)
	}
	config.PrivateKey = priv

	knownDevices, err := loadKnownDevices()
	if err != nil {
		log.Println("Warning: error while loading known devices:", err)
	}
	config.KnownDevices = knownDevices

	battery := plugin.NewBattery()
	ping := plugin.NewPing()
	notification := plugin.NewNotification()
	mprisPlugin := plugin.NewMpris()
	telephony := plugin.NewTelephony()

	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}

	notifier, err := notify.New(conn)
	if err != nil {
		panic(err)
	}

	go (func() {
		notificationsMap := map[string]int{}
		var callNotification int

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
				// TODO: remove notification when charging
			case event := <-notification.Incoming:
				log.Println("Notification:", event.Device.Name, event.NotificationBody)

				id, exists := notificationsMap[event.NotificationBody.Id]

				if event.IsCancel {
					if exists {
						notifier.CloseNotification(id)
					}
					break
				}

				n := newNotification()
				n.AppIcon = getDeviceIcon(event.Device)
				n.Summary = "Notification from "+event.AppName+" on "+event.Device.Name
				n.Body = event.Ticker
				if exists {
					n.ReplacesID = uint32(id)
				}
				newId, _ := notifier.SendNotification(n)

				notificationsMap[event.NotificationBody.Id] = int(newId)

				// TODO: wait for notification dismiss and send message to remote
			case event := <-mprisPlugin.Incoming:
				log.Println("Mpris:", event.Device.Name, event.MprisBody)

				if event.RequestPlayerList {
					names, err := mpris.List(conn)
					if err != nil {
						log.Println("Warning: cannot list available MPRIS players", err)
						break
					}

					mprisPlugin.SendPlayerList(event.Device, names)
				}

				if event.Player != "" {
					player := mpris.New(conn, event.Player)

					event.RequestNowPlaying = true
					switch event.Action {
					case "Next":
						player.Next()
					case "Previous":
						player.Previous()
					case "Pause":
						player.Pause()
					case "PlayPause":
						player.PlayPause()
					case "Stop":
						player.Stop()
					case "Play":
						player.Play()
					default:
						event.RequestNowPlaying = false
					}

					if event.SetVolume != 0 {
						player.SetVolume(float64(event.SetVolume) / 100)
						event.RequestVolume = true
					}

					if event.RequestNowPlaying || event.RequestVolume {
						reply := &plugin.MprisBody{}
						if event.RequestNowPlaying {
							metadata := player.GetMetadata()
							reply.NowPlaying = metadata["xesam:title"].String()
							reply.IsPlaying = (player.GetPlaybackStatus() == "Playing")
							reply.Length = float64(metadata["mpris:length"].Value().(int64)) / 1000
							reply.Pos = float64(player.GetPosition()) / 1000
						}
						if event.RequestVolume {
							reply.Volume = int(player.GetVolume() * 100)
						}
						event.Device.Send(plugin.MprisType, reply)
					}
				}
			case event := <-telephony.Incoming:
				log.Println("Telephony:", event.Device.Name, event.TelephonyBody)

				contactName := event.PhoneNumber
				if contactName == "" {
					contactName = event.PhoneNumber
				}

				if event.TelephonyBody.Event == plugin.TelephonySms {
					n := newNotification()
					n.AppIcon = getDeviceIcon(event.Device)
					n.Hints = map[string]dbus.Variant{
						"category": dbus.MakeVariant("im.received"),
					}
					n.Summary = "SMS from "+contactName+" on "+event.Device.Name
					n.Body = event.MessageBody
					notifier.SendNotification(n)
					break
				}

				if event.IsCancel {
					if callNotification != 0 {
						notifier.CloseNotification(callNotification)
					}
					break
				}

				n := newNotification()
				n.Hints = map[string]dbus.Variant{
					"category": dbus.MakeVariant("im"),
				}
				if callNotification != 0 {
					n.ReplacesID = uint32(callNotification)
				}

				switch event.TelephonyBody.Event {
				case plugin.TelephonyRinging:
					n.AppIcon = "call-start"
					n.Summary = "Call from "+contactName+" on "+event.Device.Name
				case plugin.TelephonyTalking:
					n.AppIcon = "call-start"
					n.Summary = "Calling "+contactName+" on "+event.Device.Name
				case plugin.TelephonyMissedCall:
					n.AppIcon = "call-stop"
					n.Summary = "Missed call from "+contactName+" on "+event.Device.Name
				}

				id, _ := notifier.SendNotification(n)
				callNotification = int(id)
			}
		}
	})()

	hdlr := plugin.NewHandler()
	hdlr.Register(battery)
	hdlr.Register(ping)
	hdlr.Register(notification)
	hdlr.Register(mprisPlugin)
	hdlr.Register(telephony)

	e := engine.New(hdlr, config)

	go (func() {
		devices := map[string]*network.Device{}
		notifications := map[string]int{}

		closed := notifier.NotificationClosed()
		actions := notifier.ActionInvoked()

		defer (func() {
			// Close all notifications
			for _, id := range notifications {
				notifier.CloseNotification(id)
			}
		})()

		getDeviceFromNotification := func (notificationId int) *network.Device {
			for deviceId, id := range notifications {
				if id == notificationId {
					if device, ok := devices[deviceId]; ok {
						return device
					} else {
						return nil
					}
				}
			}
			return nil
		}

		deviceAvailable := func(device *network.Device) {
			n := newNotification()
			n.AppIcon = getDeviceIcon(device)
			n.Summary = device.Name
			n.Body = "New device available"
			n.Hints = map[string]dbus.Variant{
				"category": dbus.MakeVariant("device"),
			}
			n.Actions = []string{"pair", "Pair device"}
			id, _ := notifier.SendNotification(n)

			notifications[device.Id] = int(id)
		}

		deviceRequestsPairing := func(device *network.Device) {
			n := newNotification()
			n.AppIcon = getDeviceIcon(device)
			n.Summary = device.Name
			n.Body = "New pair request"
			n.Hints = map[string]dbus.Variant{
				"category": dbus.MakeVariant("device"),
			}
			n.Actions = []string{"pair", "Accept", "unpair", "Reject"}
			id, _ := notifier.SendNotification(n)

			notifications[device.Id] = int(id)
		}

		deviceConnected := func(device *network.Device) {
			n := newNotification()
			n.AppIcon = getDeviceIcon(device)
			n.Summary = device.Name
			n.Body = "Device connected"
			n.Hints = map[string]dbus.Variant{
				"resident": dbus.MakeVariant(true),
				"category": dbus.MakeVariant("device.added"),
			}
			id, _ := notifier.SendNotification(n)

			notifications[device.Id] = int(id)
		}

		for {
			select {
			case device := <-e.Joins:
				if device.Id == "" {
					continue
				}

				devices[device.Id] = device

				if device.Paired {
					deviceConnected(device)
				} else {
					deviceAvailable(device)
				}
			case device := <-e.RequestsPairing:
				if id, ok := notifications[device.Id]; ok {
					notifier.CloseNotification(id)
				}

				deviceRequestsPairing(device)
			case device := <-e.Paired:
				if id, ok := notifications[device.Id]; ok {
					notifier.CloseNotification(id)
				}

				err := saveKnownDevices(config.KnownDevices)
				if err != nil {
					log.Println("Cannot save known devices:", err)
				}

				deviceConnected(device)
			case device := <-e.Unpaired:
				if id, ok := notifications[device.Id]; ok {
					notifier.CloseNotification(id)
				}
			case device := <-e.Leaves:
				if id, ok := notifications[device.Id]; ok {
					notifier.CloseNotification(id)
				}
				if _, ok := devices[device.Id]; ok {
					delete(devices, device.Id)
				}
			case signal := <-actions:
				device := getDeviceFromNotification(int(signal.Id))
				if device == nil {
					continue
				}

				log.Println(device.Name, signal.ActionKey)

				switch signal.ActionKey {
				case "pair":
					err := e.PairDevice(device)
					if err != nil {
						log.Println("Cannot pair device:", err)
					}
				case "unpair":
					err := e.UnpairDevice(device)
					if err != nil {
						log.Println("Cannot unpair device:", err)
					}
				}
			case signal := <-closed:
				device := getDeviceFromNotification(int(signal.Id))
				if device != nil {
					log.Println(device.Name, signal.Reason)

					if signal.Reason == notify.ReasonDismissedByUser {
						//device.Close()
					}
				}
			}
		}
	})()

	e.Listen()
}
