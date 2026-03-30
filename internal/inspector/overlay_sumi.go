package inspector

import (
	sumi "github.com/tomyan/sumi/runtime/prelude"
)

type OverlayProps struct {
	Visible   *sumi.Signal[bool]
	Connected *sumi.Signal[bool]
}

func NewOverlay(props OverlayProps) *sumi.Component {
	visible := props.Visible
	connected := props.Connected

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
					Kind:      sumi.KindBox,
					Padding:   sumi.ParsePadding("1 2"),
					Border:    "single",
					MinWidth:  56,
					Position:  "fixed",
					CursorCol: -1,
					CursorRow: -1,
					Children: func() []*sumi.Input {
						var cs []*sumi.Input
						cs = append(cs, &sumi.Input{
							Kind:      sumi.KindBox,
							CursorCol: -1,
							CursorRow: -1,
							Style: sumi.Style{
								Bold: true,
							},
							Children: []*sumi.Input{
								{
									Kind:    sumi.KindText,
									Content: "Connection",
								},
							},
						})
						cs = append(cs, &sumi.Input{
							Kind:    sumi.KindText,
							Content: " ",
						})
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
										Content: "● Connected",
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
