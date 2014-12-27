package ui

import "github.com/conformal/gotk3/gtk"

// A StatusBar contains the status bar UI elements.
type StatusBar struct {
	CmdStatus      *gtk.Label
	LocationStatus *gtk.Label
	container      gtk.Container
}

// SetLocationMarkup sets the text markup of the location.
func (s *StatusBar) SetLocationMarkup(label string) {
	GlibMainContextInvoke(s.LocationStatus.SetMarkup, label)
}

// SetCmdLabel sets the text of the command status.
func (s *StatusBar) SetCmdLabel(label string) {
	GlibMainContextInvoke(s.CmdStatus.SetLabel, label)
}
