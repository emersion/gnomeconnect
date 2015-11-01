package utils

import (
	"encoding/json"
	"github.com/allan-simon/go-singleinstance"
	"github.com/emersion/go-kdeconnect/crypto"
	"github.com/emersion/go-kdeconnect/engine"
	"io/ioutil"
	"os"
)

func GetConfigDir() (configDir string, err error) {
	configHomeDir := os.Getenv("XDG_CONFIG_HOME")
	if configHomeDir == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return
		}
		configHomeDir = homeDir + "/.config"
	}

	configDir = configHomeDir + "/gnomeconnect"
	err = os.MkdirAll(configDir, 0755)
	return
}

func CreateLockFile() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	_, err = singleinstance.CreateLockFile(configDir + "/server.lock")
	return err
}

func LoadPrivateKey() (priv *crypto.PrivateKey, err error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return
	}

	priv = &crypto.PrivateKey{}

	privateKeyFile := configDir + "/private.pem"
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

func LoadKnownDevices() (knownDevices []*engine.KnownDevice, err error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return
	}

	knownDevicesFile, err := os.Open(configDir + "/known-devices.json")
	if err != nil {
		return
	}

	dec := json.NewDecoder(knownDevicesFile)
	err = dec.Decode(&knownDevices)
	return
}

func SaveKnownDevices(knownDevices []*engine.KnownDevice) (err error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return
	}

	knownDevicesFile, err := os.Create(configDir + "/known-devices.json")
	if err != nil {
		return
	}

	enc := json.NewEncoder(knownDevicesFile)
	err = enc.Encode(knownDevices)
	return
}
