package gui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	unknownMAC            = "--"
	progressDelayInterval = 12090 * time.Millisecond
)

type installSession struct {
	mv *mainView
	ip string

	ctx    context.Context
	cancel context.CancelFunc

	mu              sync.Mutex
	mac             string
	status          string
	logLines        []string
	finished        bool
	userCancelled   bool
	unlockStartOnce sync.Once
	unlockStartFn   func()

	ipLabel   *widget.Label
	macLabel  *widget.Label
	progress  *widget.ProgressBar
	logBtn    *widget.Button
	actionBtn *widget.Button
	row       *fyne.Container
}

func (mv *mainView) newInstallSession(ip string) *installSession {
	ctx, cancel := context.WithCancel(context.Background())
	session := &installSession{
		mv:     mv,
		ip:     ip,
		ctx:    ctx,
		cancel: cancel,
		mac:    unknownMAC,
		status: "Running",
	}

	session.ipLabel = widget.NewLabel(ip)
	session.ipLabel.TextStyle = fyne.TextStyle{Bold: true}

	session.macLabel = widget.NewLabel(session.macWithStatus())

	session.progress = widget.NewProgressBar()
	session.progress.SetValue(0)

	session.logBtn = widget.NewButton("Logs", func() {
		session.showLogs()
	})

	session.actionBtn = widget.NewButton("Cancel", func() {
		session.confirmCancel()
	})

	top := container.NewBorder(nil, nil, container.NewHBox(session.ipLabel, session.macLabel), container.NewHBox(session.logBtn, session.actionBtn))
	session.row = container.NewBorder(widget.NewSeparator(), widget.NewSeparator(), nil, nil, container.NewVBox(top, session.progress))

	mv.runOnUI(func() {
		mv.sessions = append(mv.sessions, session)
		mv.sessionsBox.Add(session.row)
		mv.sessionsBox.Refresh()
	})

	return session
}

func (s *installSession) setStartUnlocker(fn func()) {
	s.unlockStartFn = fn
}

func (s *installSession) unlockStart() {
	s.unlockStartOnce.Do(func() {
		if s.unlockStartFn != nil {
			s.unlockStartFn()
		}
	})
}

func (s *installSession) appendLog(line, replaceIdentifier string) {
	formatted := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line)

	s.mu.Lock()
	var labelUpdate string
	if replaceIdentifier != "" && len(s.logLines) > 0 {
		replaced := false
		for i, existing := range s.logLines {
			if strings.Contains(existing, replaceIdentifier) {
				s.logLines[i] = formatted
				replaced = true
			}
		}
		if !replaced {
			s.logLines = append(s.logLines, formatted)
		}
	} else {
		s.logLines = append(s.logLines, formatted)
	}

	if strings.Contains(line, "Device MAC address:") {
		if mac := parseMACFromLog(line); mac != "" {
			s.mac = mac
			labelUpdate = s.macWithStatusLocked()
		}
	}
	s.mu.Unlock()

	if labelUpdate != "" {
		s.mv.runOnUI(func() {
			s.macLabel.SetText(labelUpdate)
		})
	}
}

func (s *installSession) setStatus(status string) {
	s.mu.Lock()
	s.status = status
	labelText := s.macWithStatusLocked()
	s.mu.Unlock()

	s.mv.runOnUI(func() {
		s.macLabel.SetText(labelText)
	})
}

func parseMACFromLog(line string) string {
	parts := strings.SplitN(line, "Device MAC address:", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (s *installSession) macWithStatusLocked() string {
	status := s.status
	if status == "" {
		status = "Running"
	}
	return fmt.Sprintf("MAC: %s  [%s]", s.mac, status)
}

func (s *installSession) macWithStatus() string {
	s.mu.Lock()
	text := s.macWithStatusLocked()
	s.mu.Unlock()
	return text
}

func (s *installSession) showLogs() {
	text := s.logSnapshot()
	s.mv.runOnUI(func() {
		entry := widget.NewMultiLineEntry()
		entry.SetText(text)
		entry.OnChanged = func(value string) {
			if value != text {
				entry.SetText(text)
			}
		}
		entry.Wrapping = fyne.TextWrapWord
		entry.SetMinRowsVisible(18)

		scroll := container.NewVScroll(entry)
		scroll.SetMinSize(fyne.NewSize(700, 400))

		dialog.NewCustom("Logs for "+s.ip, "Close", scroll, s.mv.window).Show()
	})
}

// func (s *installSession) updateProgress(value, _ float64) {
// 	s.mv.runOnUI(func() {
// 		s.progress.SetValue(value)
// 	})
// }

func (s *installSession) updateProgress(value float64, targetValue float64) {
	s.mv.runOnUI(func() {
		s.progress.SetValue(value)
	})
	if value >= targetValue {
		return
	}
	go func(startValue float64, target float64) {
		current := startValue
		for {
			time.Sleep(progressDelayInterval)
			s.mv.runOnUI(func() {
				actual := s.progress.Value
				if actual != current {
					return
				}
				if current >= target {
					s.progress.SetValue(target)
					return
				}
				current += 0.01
				if current > target {
					current = target
				}
				s.progress.SetValue(current)
			})
		}
	}(value, targetValue)
}

func (s *installSession) reportSuccess() {
	s.unlockStart()
	s.appendLog("Installation completed successfully", "")
	s.mv.runOnUI(func() {
		s.progress.SetValue(1)
	})
	s.setStatus("Completed")
	s.finish()
}

func (s *installSession) reportFailure(err error) {
	s.unlockStart()
	s.appendLog("Error: "+err.Error(), "")
	s.setStatus("Failed")
	s.finish()
}

func (s *installSession) reportCancellation() {
	s.unlockStart()
	if s.wasUserCancelled() {
		s.appendLog("Installation cancelled by user", "")
		s.setStatus("Cancelled by user")
	} else {
		s.appendLog("Installation cancelled", "")
		s.setStatus("Cancelled")
	}
	s.finish()
}

func (s *installSession) confirmCancel() {
	if s.isFinished() {
		s.mv.removeSession(s)
		return
	}

	dialog.NewConfirm(
		"Cancel installation?",
		"Cancelling will stop the running installation for this device.",
		func(ok bool) {
			if !ok {
				return
			}
			if s.markUserCancelled() {
				s.appendLog("Cancellation requested by user", "")
				s.setStatus("Cancelling")
				s.mv.runOnUI(func() {
					s.actionBtn.Disable()
					s.actionBtn.SetText("Cancelling...")
				})
				s.cancel()
			}
		},
		s.mv.window,
	).Show()
}

func (s *installSession) finish() {
	s.mu.Lock()
	if s.finished {
		s.mu.Unlock()
		return
	}
	s.finished = true
	s.mu.Unlock()

	s.unlockStart()
	s.cancel()

	s.mv.runOnUI(func() {
		s.actionBtn.Enable()
		s.actionBtn.SetText("Remove")
		s.actionBtn.OnTapped = func() {
			s.mv.removeSession(s)
		}
		s.actionBtn.Refresh()
		s.macLabel.SetText(s.macWithStatus())
	})
}

func (s *installSession) markUserCancelled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finished {
		return false
	}
	if s.userCancelled {
		return false
	}
	s.userCancelled = true
	return true
}

func (s *installSession) wasUserCancelled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userCancelled
}

func (s *installSession) isFinished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finished
}

func (s *installSession) logSnapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.logLines, "\n")
}

func (mv *mainView) removeSession(target *installSession) {
	mv.runOnUI(func() {
		for i, session := range mv.sessions {
			if session == target {
				mv.sessions = append(mv.sessions[:i], mv.sessions[i+1:]...)
				break
			}
		}
		mv.sessionsBox.Remove(target.row)
		mv.sessionsBox.Refresh()
	})
}
