package inspector

import (
	sumi "github.com/tomyan/sumi/runtime/prelude"
)

type OverlayProps struct {
	Visible        *sumi.Signal[bool]
	Connected      *sumi.Signal[bool]
	PageTitle      *sumi.Signal[string]
	PageURL        *sumi.Signal[string]
	TargetID       *sumi.Signal[string]
	BrowserVersion *sumi.Signal[string]
}

func NewOverlay(props OverlayProps) *sumi.Component {
	visible := props.Visible
	connected := props.Connected
	pageTitle := props.PageTitle
	pageURL := props.PageURL
	targetID := props.TargetID
	browserVersion := props.BrowserVersion

	root := &sumi.Input{
		Kind:      sumi.KindBox,
		Direction: "column",
		CursorCol: -1,
		CursorRow: -1,
	}

	sumi.Effect(func() {
		root.Children = func() []*sumi.Input {
			var cs []*sumi.Input
			if visible.Get() {
				cs = append(cs, &sumi.Input{
					Kind:        sumi.KindBox,
					Padding:     sumi.ParsePadding("1 2"),
					Border:      "single",
					BorderTitle: "\"Connection\"",
					MinWidth:    56,
					Position:    "fixed",
					CursorCol:   -1,
					CursorRow:   -1,
					Children: func() []*sumi.Input {
						var cs []*sumi.Input
						if connected.Get() {
							cs = append(cs, &sumi.Input{
								Kind:      sumi.KindBox,
								CursorCol: -1,
								CursorRow: -1,
								Style: sumi.Style{
									FG:   sumi.Color{IsRGB: true, R: 80, G: 250, B: 123},
									Bold: true,
								},
								Children: []*sumi.Input{
									{
										Kind:    sumi.KindText,
										Content: sumi.Sprintf("● Connected to %v", pageTitle.Get()),
									},
								},
							})
							cs = append(cs, &sumi.Input{
								Kind:      sumi.KindBox,
								CursorCol: -1,
								CursorRow: -1,
								Style: sumi.Style{
									Dim: true,
								},
								Children: []*sumi.Input{
									{
										Kind:    sumi.KindText,
										Content: sumi.Sprintf("%v", pageURL.Get()),
									},
								},
							})
							cs = append(cs, &sumi.Input{
								Kind:      sumi.KindBox,
								CursorCol: -1,
								CursorRow: -1,
								Style: sumi.Style{
									Dim: true,
								},
								Children: []*sumi.Input{
									{
										Kind:    sumi.KindText,
										Content: sumi.Sprintf("Target: %v  Browser: %v", targetID.Get(), browserVersion.Get()),
									},
								},
							})
						} else {
							cs = append(cs, &sumi.Input{
								Kind:      sumi.KindBox,
								CursorCol: -1,
								CursorRow: -1,
								Style: sumi.Style{
									FG:   sumi.Color{IsRGB: true, R: 255, G: 85, B: 85},
									Bold: true,
								},
								Children: []*sumi.Input{
									{
										Kind:    sumi.KindText,
										Content: "● Disconnected — reconnecting every 2s",
									},
								},
							})
							cs = append(cs, &sumi.Input{
								Kind:      sumi.KindBox,
								CursorCol: -1,
								CursorRow: -1,
								Style: sumi.Style{
									Dim: true,
								},
								Children: []*sumi.Input{
									{
										Kind:    sumi.KindText,
										Content: sumi.Sprintf("Last: %v — %v", pageTitle.Get(), pageURL.Get()),
									},
								},
							})
						}
						cs = append(cs, &sumi.Input{
							Kind:    sumi.KindText,
							Content: " ",
						})
						cs = append(cs, &sumi.Input{
							Kind:      sumi.KindBox,
							Padding:   sumi.ParsePadding("1 0 0 0"),
							BorderTop: "single",
							CursorCol: -1,
							CursorRow: -1,
							Style: sumi.Style{
								Dim: true,
							},
							Children: []*sumi.Input{
								{
									Kind:    sumi.KindText,
									Content: "Esc Close",
								},
							},
						})
						return cs
					}(),
				})
			}
			return cs
		}()
	})

	return &sumi.Component{
		Tree: root,
	}
}
