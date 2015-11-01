package ui

import (
	"github.com/conformal/gotk3/gtk"
	"github.com/emersion/gnomeconnect/utils"
	"github.com/emersion/go-kdeconnect/engine"
	"github.com/emersion/go-kdeconnect/network"
	"github.com/emersion/go-kdeconnect/plugin"
	"log"
)

type PluginCollection struct {
	Sftp *plugin.Sftp
}

const (
	sidebarWidth = 200
)

type Ui struct {
	win            *gtk.Window
	selectedDevice *network.Device
	devices        map[string]*network.Device
	engine         *engine.Engine
	plugins        *PluginCollection

	devicesList *gtk.ListBox
	devicesRows map[string]*gtk.ListBoxRow

	deviceBox         *gtk.Box
	deviceNameLabel   *gtk.Label
	deviceStatusLabel *gtk.Label
	deviceIcon        *gtk.Image
	pairBtn           *gtk.Button
	browseBtn         *gtk.Button

	Available       chan *network.Device
	Unavailable     chan *network.Device
	RequestsPairing chan *network.Device
	Connected       chan *network.Device
	Disconnected    chan *network.Device

	Quit chan bool
}

func (ui *Ui) Raise() {
	ui.win.Present()
}

func (ui *Ui) SelectDevice(device *network.Device) {
	ui.addDevice(device)

	if row, ok := ui.devicesRows[device.Id]; ok {
		ui.devicesList.SelectRow(row)
	}
}

func (ui *Ui) selectDevice(device *network.Device) {
	ui.selectedDevice = device

	ui.deviceBox.SetVisible(device != nil)

	if device == nil {
		return
	}

	ui.deviceNameLabel.SetMarkup("<big>" + device.Name + "</big>")
	ui.deviceIcon.SetFromIconName(utils.GetDeviceIcon(device), gtk.ICON_SIZE_DIALOG)
	ui.browseBtn.SetVisible(device.Paired)

	if device.Paired {
		ui.deviceStatusLabel.SetText("Device connected")
		ui.pairBtn.SetLabel("Unpair")
	} else {
		ui.deviceStatusLabel.SetText("Device available")
		ui.pairBtn.SetLabel("Pair")
	}
}

func (ui *Ui) updateDevicesList() {
	if ui.devicesRows != nil {
		for _, row := range ui.devicesRows {
			row.Destroy()
		}
	}
	ui.devicesRows = map[string]*gtk.ListBoxRow{}

	for _, device := range ui.devices {
		row, _ := gtk.ListBoxRowNew()
		ui.devicesList.Add(row)
		l, _ := gtk.LabelNew(device.Name)
		l.Set("xalign", 0)
		l.SetPadding(20, 15)
		row.Add(l)

		ui.devicesRows[device.Id] = row
	}

	ui.devicesList.ShowAll()
}

func (ui *Ui) initDevicesList() *gtk.ScrolledWindow {
	scroller, _ := gtk.ScrolledWindowNew(nil, nil)
	scroller.SetSizeRequest(sidebarWidth, -1)

	list, _ := gtk.ListBoxNew()
	scroller.Add(list)
	ui.devicesList = list

	list.Connect("row-selected", func(box *gtk.ListBox, row *gtk.ListBoxRow) {
		index := row.GetIndex()

		for deviceId, r := range ui.devicesRows {
			if index == r.GetIndex() {
				ui.selectDevice(ui.devices[deviceId])
				return
			}
		}
	})

	return scroller
}

func (ui *Ui) initDeviceView() *gtk.Box {
	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	ui.deviceBox = vbox

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	vbox.PackStart(hbox, false, true, 20)

	img, _ := gtk.ImageNew()
	hbox.PackStart(img, false, true, 0)
	ui.deviceIcon = img

	nameBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	hbox.PackStart(nameBox, true, true, 0)

	l, _ := gtk.LabelNew("")
	l.Set("xalign", 0)
	nameBox.PackStart(l, true, true, 0)
	ui.deviceNameLabel = l

	l, _ = gtk.LabelNew("")
	l.Set("xalign", 0)
	nameBox.PackStart(l, true, true, 0)
	ui.deviceStatusLabel = l

	browseBtn, _ := gtk.ButtonNewFromIconName("document-open-symbolic", gtk.ICON_SIZE_BUTTON)
	hbox.PackStart(browseBtn, false, false, 0)
	ui.browseBtn = browseBtn

	browseBtn.Connect("clicked", func() {
		log.Println("Browse device", ui.selectedDevice)
		ui.plugins.Sftp.SendStartBrowsing(ui.selectedDevice)
	})

	pairBtn, _ := gtk.ButtonNew()
	hbox.PackStart(pairBtn, false, false, 5)
	ui.pairBtn = pairBtn

	pairBtn.Connect("clicked", func() {
		log.Println("Pair/unpair device", ui.selectedDevice)

		if ui.selectedDevice.Paired {
			ui.engine.UnpairDevice(ui.selectedDevice)
		} else {
			ui.engine.PairDevice(ui.selectedDevice)
		}
	})

	return vbox
}

func (ui *Ui) initTitlebar() *gtk.Box {
	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	headerbar, _ := gtk.HeaderBarNew()
	headerbar.SetTitle("Devices")
	headerbar.SetSizeRequest(sidebarWidth, -1)
	hbox.PackStart(headerbar, false, true, 0)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	hbox.PackStart(sep, false, true, 0)

	headerbar, _ = gtk.HeaderBarNew()
	headerbar.SetTitle("GNOMEConnect")
	headerbar.SetShowCloseButton(true)
	hbox.PackEnd(headerbar, true, true, 0)

	return hbox
}

func (ui *Ui) init() {
	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GNOMEConnect")
	win.SetDefaultSize(800, 600)
	win.Connect("destroy", func() {
		gtk.MainQuit()
		ui.Quit <- true
	})
	ui.win = win

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	win.Add(hbox)

	scroller := ui.initDevicesList()
	hbox.PackStart(scroller, false, true, 0)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	hbox.PackStart(sep, false, true, 0)

	box := ui.initDeviceView()
	hbox.PackEnd(box, true, true, 0)

	titlebar := ui.initTitlebar()
	win.SetTitlebar(titlebar)

	win.ShowAll()
	ui.selectDevice(nil)
}

func (ui *Ui) addDevice(device *network.Device) {
	if _, ok := ui.devices[device.Id]; !ok {
		ui.devices[device.Id] = device
		ui.updateDevicesList()
	}
}

func (ui *Ui) removeDevice(device *network.Device) {
	if ui.selectedDevice != nil && ui.selectedDevice.Id == device.Id {
		ui.selectDevice(nil)
	}

	if _, ok := ui.devices[device.Id]; ok {
		delete(ui.devices, device.Id)
		ui.updateDevicesList()
	}
}

func (ui *Ui) listen() {
	for {
		select {
		case device := <-ui.Available:
			ui.addDevice(device)
		case device := <-ui.Unavailable:
			ui.removeDevice(device)
		case device := <-ui.Connected:
			ui.addDevice(device)
		case device := <-ui.Disconnected:
			ui.addDevice(device)
		}

		ui.selectDevice(ui.selectedDevice)
	}
}

func New(engine *engine.Engine, plugins *PluginCollection) *Ui {
	gtk.Init(nil)

	ui := &Ui{
		engine:  engine,
		plugins: plugins,
		devices: map[string]*network.Device{},

		Available:       make(chan *network.Device),
		Unavailable:     make(chan *network.Device),
		RequestsPairing: make(chan *network.Device),
		Connected:       make(chan *network.Device),
		Disconnected:    make(chan *network.Device),

		Quit: make(chan bool),
	}

	ui.init()
	go gtk.Main()
	go ui.listen()

	return ui
}
