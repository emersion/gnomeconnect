package utils

import (
	"github.com/emersion/go-kdeconnect/network"
)

func GetDeviceIcon(device *network.Device) string {
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
