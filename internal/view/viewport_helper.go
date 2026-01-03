package view

import "charm.land/bubbles/v2/viewport"

type ViewportState struct {
	Model viewport.Model
	Ready bool
}

func (vs *ViewportState) SetSize(width, height int) {
	if !vs.Ready {
		vs.Model = viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
		vs.Ready = true
	} else {
		vs.Model.SetWidth(width)
		vs.Model.SetHeight(height)
	}
}
