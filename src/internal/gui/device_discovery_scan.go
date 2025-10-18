package gui

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"wago-init/internal/install"

	"fyne.io/fyne/v2/widget"
)

func (mv *mainView) runDeviceScan(
	ctx context.Context,
	ips []string,
	devices *[]discoveredDevice,
	table *widget.Table,
	status *widget.Label,
	scanBtn *widget.Button,
	selected *bool,
) {
	total := len(ips)

	var (
		processedCount int64
		foundCount     int64
		wg             sync.WaitGroup
		sem            = make(chan struct{}, deviceScanConcurrency)
	)

	updateStatus := func(message string) {
		mv.runOnUI(func() {
			if selected != nil && *selected {
				return
			}
			status.SetText(message)
		})
	}

	appendDevice := func(ip, mac string, processed, found int64) {
		mv.runOnUI(func() {
			*devices = append(*devices, discoveredDevice{IP: ip, MAC: mac})
			if selected == nil || !*selected {
				status.SetText(fmt.Sprintf(
					"Found %d device(s). Scanning (%d/%d)...",
					found,
					processed,
					total,
				))
			}
			table.Refresh()
		})
	}

Loop:
	for _, ip := range ips {
		select {
		case <-ctx.Done():
			break Loop
		default:
		}

		select {
		case <-ctx.Done():
			break Loop
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(ip string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			if ctx.Err() != nil {
				return
			}

			mac, allowed, err := install.DiscoverDeviceMAC(ip)
			if err != nil {
				processed := atomic.AddInt64(&processedCount, 1)
				updateStatus(fmt.Sprintf(
					"Scanning (%d/%d)... last error: %v",
					processed,
					total,
					err,
				))
				return
			}

			processed := atomic.AddInt64(&processedCount, 1)
			if allowed {
				found := atomic.AddInt64(&foundCount, 1)
				appendDevice(ip, mac, processed, found)
			} else {
				updateStatus(fmt.Sprintf("Scanning (%d/%d)...", processed, total))
			}
		}(ip)
	}

	wg.Wait()

	mv.runOnUI(func() {
		scanBtn.Enable()
		if selected != nil && *selected {
			return
		}
		if ctx.Err() == context.Canceled {
			status.SetText("Scan cancelled.")
			return
		}
		status.SetText(fmt.Sprintf(
			"Scan finished. Found %d device(s).",
			atomic.LoadInt64(&foundCount),
		))
	})
}
