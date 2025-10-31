package gui

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type passwordResponse struct {
	password string
	ok       bool
}

func passwordPromtFunc(parent fyne.Window) func() (string, bool) {
	return func() (string, bool) {
		resultCh := make(chan passwordResponse, 1)

		fyne.Do(func() {
			entry := widget.NewPasswordEntry()
			form := dialog.NewForm(
				"Please enter the current root SSH Password of the device",
				"Connect",
				"Cancel",
				[]*widget.FormItem{
					widget.NewFormItem("Password", entry),
				},
				func(ok bool) {
					if ok {
						resultCh <- passwordResponse{password: entry.Text, ok: true}
					} else {
						resultCh <- passwordResponse{ok: false}
					}
				},
				parent,
			)
			form.Resize(fyne.NewSize(400, 160))
			form.SetOnClosed(func() {
				select {
				case resultCh <- passwordResponse{ok: false}:
				default:
				}
			})
			form.Show()
		})

		res := <-resultCh
		return res.password, res.ok
	}
}

func newPasswordPromtFunc(parent fyne.Window) func(*installSession) (string, bool) {
	return func(session *installSession) (string, bool) {
		resultCh := make(chan passwordResponse, 1)

		fyne.Do(func() {
			entry := widget.NewPasswordEntry()
			generateBtn := widget.NewButton("Generate Password", func() {
				pwd, err := generateSecurePassword(20)
				if err != nil {
					dialog.ShowError(err, parent)
					return
				}
				entry.SetText(pwd)
				if clip := GetClipboard(parent); clip != nil {
					clip.SetContent(pwd)
				}
			})
			copyMacBtn := widget.NewButton("MAC-Address", func() {
				if session == nil {
					return
				}
				if mac := session.macValue(); mac != "" {
					if clip := GetClipboard(parent); clip != nil {
						clip.SetContent(mac)
					}
				}
			})
			copySerialBtn := widget.NewButton("Serial-Number", func() {
				if session == nil {
					return
				}
				if serial := session.serialValue(); serial != "" {
					if clip := GetClipboard(parent); clip != nil {
						clip.SetContent(serial)
					}
				}
			})
			copyRow := container.NewHBox(copySerialBtn, copyMacBtn)
			form := dialog.NewForm(
				"New Password required.\nEnter a new secure Password for the device",
				"Change Password",
				"Cancel",
				[]*widget.FormItem{
					widget.NewFormItem("Copy to Clipboard", copyRow),
					widget.NewFormItem("Password", entry),
					widget.NewFormItem("", generateBtn),
				},
				func(ok bool) {
					if ok {
						resultCh <- passwordResponse{password: entry.Text, ok: true}
					} else {
						resultCh <- passwordResponse{ok: false}
					}
				},
				parent,
			)
			form.Resize(fyne.NewSize(400, 160))
			form.SetOnClosed(func() {
				select {
				case resultCh <- passwordResponse{ok: false}:
				default:
				}
			})
			form.Show()
		})

		res := <-resultCh
		return res.password, res.ok
	}
}

const (
	upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowerChars   = "abcdefghijklmnopqrstuvwxyz"
	digitChars   = "0123456789"
	specialChars = "!@#$%^&*()-_=+[]{}<>?/|~"
)

func generateSecurePassword(length int) (string, error) {
	if length < 4 {
		return "", fmt.Errorf("password length must be at least 4 characters")
	}

	charSets := []string{upperChars, lowerChars, digitChars, specialChars}
	allChars := upperChars + lowerChars + digitChars + specialChars

	password := make([]byte, length)

	for i, set := range charSets {
		idx, err := cryptoRandIndex(len(set))
		if err != nil {
			return "", err
		}
		password[i] = set[idx]
	}

	for i := len(charSets); i < length; i++ {
		idx, err := cryptoRandIndex(len(allChars))
		if err != nil {
			return "", err
		}
		password[i] = allChars[idx]
	}

	if err := shuffleBytes(password); err != nil {
		return "", err
	}

	return string(password), nil
}

func cryptoRandIndex(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("invalid max value %d", max)
	}

	val, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random index: %w", err)
	}
	return int(val.Int64()), nil
}

func shuffleBytes(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("failed to shuffle password: %w", err)
		}
		j := int(idx.Int64())
		data[i], data[j] = data[j], data[i]
	}
	return nil
}
