package gui

import (
	"context"
	"fmt"
	"strings"

	"wago-init/internal/fs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var searchEntry string

func BuildDeviceDiscoveryPrompt(mv *mainView) *widget.Button {
	return widget.NewButton("Device discovery", func() {
		mv.showDeviceDiscoveryDialog()
	})
}

func (mv *mainView) showDeviceDiscoveryDialog() {
	input := widget.NewEntry()
	input.SetText(searchEntry)
	input.SetPlaceHolder("192.168.42.42  or  10.0.1.*  or  172.16.1.0/25  or  10.2.1.20 - 10.2.1.60")

	networkOptions := discoverHostNetworks()
	networkSelect := widget.NewSelect(networkOptions, func(value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		input.SetText(value)
	})
	if len(networkOptions) == 0 {
		networkSelect.PlaceHolder = "No networks found"
		networkSelect.Disable()
	} else {
		networkSelect.PlaceHolder = "Local networks"
	}

	status := widget.NewLabel("Idle")
	// Use pointer to mainView's cache for device list
	devices := &mv.deviceDiscoveryCache

	sortDevices := func() {
		sortDiscoveredDevices(devices)
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(*devices) + 1, 2
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				if id.Col == 0 {
					label.SetText("IP address")
				} else {
					label.SetText("MAC address")
				}
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.Refresh()
				return
			}

			label.TextStyle = fyne.TextStyle{}
			idx := id.Row - 1
			if idx < 0 || idx >= len(*devices) {
				label.SetText("")
				label.Refresh()
				return
			}

			device := (*devices)[idx]
			if id.Col == 0 {
				label.SetText(device.IP)
			} else {
				label.SetText(device.MAC)
			}
			label.Refresh()
		},
	)
	table.SetColumnWidth(0, 180)
	table.SetColumnWidth(1, 220)

	var (
		scanCancel context.CancelFunc
		scanBtn    *widget.Button
		selected   bool
		dlg        dialog.Dialog
	)

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 || id.Row-1 >= len(*devices) {
			return
		}
		if scanCancel != nil {
			scanCancel()
			scanCancel = nil
		}
		device := (*devices)[id.Row-1]
		mv.ipEntry.SetText(device.IP)
		if mv.configValues == nil {
			mv.configValues = fs.EnvConfig{}
		}
		mv.configValues[fs.IpAddress] = device.IP
		status.SetText("Selected " + device.IP)
		selected = true
		if scanBtn != nil {
			scanBtn.Enable()
		}
		table.UnselectAll()
		if dlg != nil {
			dlg.Hide()
			mv.window.Canvas().Focus(mv.ipEntry)
		}
	}

	scanBtn = widget.NewButton("Scan", nil)

	refreshTable := func() {
		sortDevices()
		table.Refresh()
	}

	scan := func() {
		searchEntry = input.Text
		pattern := strings.TrimSpace(input.Text)
		ips, err := expandIPPattern(pattern)
		if err != nil {
			dialog.ShowError(err, mv.window)
			return
		}
		if len(ips) == 0 {
			dialog.ShowInformation(deviceDiscoveryTitle, "No IP addresses to scan.", mv.window)
			return
		}
		if scanCancel != nil {
			scanCancel()
			scanCancel = nil
		}
		selected = false
		// Clear and reuse the cache slice
		*devices = (*devices)[:0]
		refreshTable()
		scanBtn.Disable()
		status.SetText(fmt.Sprintf("Scanning %d addresses...", len(ips)))

		ctx, cancel := context.WithCancel(context.Background())
		scanCancel = cancel

		go func() {
			mv.runDeviceScan(ctx, ips, devices, table, status, scanBtn, &selected)
			mv.runOnUI(func() {
				if scanCancel != nil {
					scanCancel = nil
				}
			})
		}()
	}

	scanBtn.OnTapped = scan

	inputRow := container.NewBorder(nil, nil, nil, networkSelect, input)
	topRow := container.NewBorder(nil, scanBtn, nil, nil, inputRow)
	layout := container.NewBorder(topRow, status, nil, nil, container.NewMax(table))

	refreshTable()

	dlg = dialog.NewCustom(deviceDiscoveryTitle, "Close", layout, mv.window)
	dlg.SetOnClosed(func() {
		if scanCancel != nil {
			scanCancel()
			scanCancel = nil
		}
	})
	dlg.Resize(fyne.NewSize(720, 500))
	dlg.Show()
}
