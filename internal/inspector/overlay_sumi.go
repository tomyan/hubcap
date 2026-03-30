package inspector

import (
	"strings"

	sumi "github.com/tomyan/sumi/runtime/prelude"
)

type OverlayProps struct {
	Visible        *sumi.Signal[bool]
	Connected      *sumi.Signal[bool]
	PageTitle      *sumi.Signal[string]
	PageURL        *sumi.Signal[string]
	TargetID       *sumi.Signal[string]
	BrowserVersion *sumi.Signal[string]
	Tabs           *sumi.Signal[[]TabInfo]
	SelectedIdx    *sumi.Signal[int]
	Filter         *sumi.Signal[string]
}

func NewOverlay(props OverlayProps) *sumi.Component {
	visible := props.Visible
	connected := props.Connected
	pageTitle := props.PageTitle
	pageURL := props.PageURL
	targetID := props.TargetID
	browserVersion := props.BrowserVersion
	tabs := props.Tabs
	selectedIdx := props.SelectedIdx
	filter := props.Filter

	termW := sumi.Env[int]("width")
	termH := sumi.Env[int]("height")

	getFilteredTabs := func() []TabInfo {
		f := strings.ToLower(filter.Get())
		if f == "" {
			return tabs.Get()
		}
		filtered := []TabInfo{}
		for _, t := range tabs.Get() {
			if strings.Contains(strings.ToLower(t.Title), f) || strings.Contains(strings.ToLower(t.URL), f) {
				filtered = append(filtered, t)
			}
		}
		return filtered
	}

	close := func() {
		visible.Set(false)
	}

	noop := func() {
	}

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
					FixedWidth:  termW.Get(),
					FixedHeight: termH.Get(),
					Position:    "fixed",
					OnClick:     close,
					CursorCol:   -1,
					CursorRow:   -1,
					Children: []*sumi.Input{
						{
							Kind:        sumi.KindBox,
							Padding:     sumi.ParsePadding("1 2"),
							Border:      "single",
							BorderTitle: "\"Connection\"",
							MinWidth:    56,
							OnClick:     noop,
							CursorCol:   -1,
							CursorRow:   -1,
							Style: sumi.Style{
								BG: sumi.Color{Name: "black"},
							},
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
									Kind:         sumi.KindBox,
									Padding:      sumi.ParsePadding("1 0 0 0"),
									BorderBottom: "single",
									CursorCol:    -1,
									CursorRow:    -1,
									Style: sumi.Style{
										Dim: true,
									},
									Children: []*sumi.Input{
										{
											Kind:    sumi.KindText,
											Content: "Tabs",
										},
									},
								})
								cs = append(cs, &sumi.Input{
									Kind:      sumi.KindBox,
									Direction: "row",
									CursorCol: -1,
									CursorRow: -1,
									Children: []*sumi.Input{
										{
											Kind:      sumi.KindBox,
											CursorCol: -1,
											CursorRow: -1,
											Style: sumi.Style{
												Bold: true,
											},
											Children: []*sumi.Input{
												{
													Kind:    sumi.KindText,
													Content: "❯ ",
												},
											},
										},
										{
											Kind:      sumi.KindBox,
											CursorCol: -1,
											CursorRow: -1,
											Style: sumi.Style{
												FG: sumi.Color{IsRGB: true, R: 241, G: 250, B: 140},
											},
											Children: []*sumi.Input{
												{
													Kind:    sumi.KindText,
													Content: sumi.Sprintf("%v", filter.Get()),
												},
											},
										},
									},
								})
								cs = append(cs, &sumi.Input{
									Kind:      sumi.KindBox,
									Overflow:  "auto",
									CursorCol: -1,
									CursorRow: -1,
									Children: func() []*sumi.Input {
										var cs []*sumi.Input
										for i, tab := range getFilteredTabs() {
											if tab.ID == targetID.Get() && i == selectedIdx.Get() {
												cs = append(cs, &sumi.Input{
													Kind:      sumi.KindBox,
													Padding:   sumi.ParsePadding("0 1"),
													CursorCol: -1,
													CursorRow: -1,
													Style: sumi.Style{
														FG:   sumi.Color{Name: "white"},
														Bold: true,
													},
													HoverStyle: sumi.Style{
														FG: sumi.Color{Name: "white"},
													},
													Children: []*sumi.Input{
														{
															Kind:    sumi.KindText,
															Content: sumi.Sprintf("→ %v", tab.Title),
														},
													},
												})
											} else {
												if tab.ID == targetID.Get() {
													cs = append(cs, &sumi.Input{
														Kind:      sumi.KindBox,
														Padding:   sumi.ParsePadding("0 1"),
														CursorCol: -1,
														CursorRow: -1,
														Style: sumi.Style{
															FG: sumi.Color{IsRGB: true, R: 80, G: 250, B: 123},
														},
														HoverStyle: sumi.Style{
															FG: sumi.Color{Name: "white"},
														},
														Children: []*sumi.Input{
															{
																Kind:    sumi.KindText,
																Content: sumi.Sprintf("→ %v", tab.Title),
															},
														},
													})
												} else {
													if i == selectedIdx.Get() {
														cs = append(cs, &sumi.Input{
															Kind:      sumi.KindBox,
															Padding:   sumi.ParsePadding("0 1"),
															CursorCol: -1,
															CursorRow: -1,
															Style: sumi.Style{
																FG:   sumi.Color{Name: "white"},
																Bold: true,
															},
															HoverStyle: sumi.Style{
																FG: sumi.Color{Name: "white"},
															},
															Children: []*sumi.Input{
																{
																	Kind:    sumi.KindText,
																	Content: sumi.Sprintf("  %v", tab.Title),
																},
															},
														})
													} else {
														cs = append(cs, &sumi.Input{
															Kind:      sumi.KindBox,
															Padding:   sumi.ParsePadding("0 1"),
															CursorCol: -1,
															CursorRow: -1,
															HoverStyle: sumi.Style{
																FG: sumi.Color{Name: "white"},
															},
															Children: []*sumi.Input{
																{
																	Kind:    sumi.KindText,
																	Content: sumi.Sprintf("  %v", tab.Title),
																},
															},
														})
													}
												}
											}
											cs[len(cs)-1].Key = sumi.Sprint(tab.ID)
										}
										return cs
									}(),
								})
								cs = append(cs, &sumi.Input{
									Kind:      sumi.KindBox,
									Direction: "row",
									Gap:       2,
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
											Content: "↑↓ Select",
										},
										{
											Kind:    sumi.KindText,
											Content: "Enter Switch",
										},
										{
											Kind:    sumi.KindText,
											Content: "f Focus",
										},
										{
											Kind:    sumi.KindText,
											Content: "n New Tab",
										},
										{
											Kind:    sumi.KindText,
											Content: "Esc Close",
										},
									},
								})
								return cs
							}(),
						},
					},
				})
			}
			return cs
		}()
	})

	return &sumi.Component{
		Tree: root,
	}
}
