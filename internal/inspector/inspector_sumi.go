package inspector

import (
	"github.com/tomyan/sumi-ui/components/console"

	sumi "github.com/tomyan/sumi/runtime/prelude"
)

type InspectorProps struct {
	Entries   *sumi.Signal[[]console.Entry]
	Prompt    *sumi.Signal[string]
	Cursor    *sumi.Signal[int]
	Connected *sumi.Signal[bool]
}

func NewInspector(props InspectorProps) *sumi.Component {
	entries := props.Entries
	prompt := props.Prompt
	cursor := props.Cursor
	connected := props.Connected

	console0 := console.NewConsole(console.ConsoleProps{
		Entries: entries,
		Prompt:  prompt,
		Cursor:  cursor,
	})

	box0 := &sumi.Input{
		Kind:         sumi.KindBox,
		Direction:    "row",
		BorderBottom: "single",
		CursorCol:    -1,
		CursorRow:    -1,
		Style: sumi.Style{
			Dim: true,
		},
	}
	root := &sumi.Input{
		Kind:      sumi.KindBox,
		Direction: "column",
		CursorCol: -1,
		CursorRow: -1,
		Children: []*sumi.Input{
			{
				Kind:      sumi.KindBox,
				CursorCol: -1,
				CursorRow: -1,
				Children: []*sumi.Input{
					box0,
					{
						Kind:      sumi.KindBox,
						FlexGrow:  1,
						CursorCol: -1,
						CursorRow: -1,
						Children: []*sumi.Input{
							console0.Tree,
						},
					},
				},
			},
		},
	}

	sumi.Effect(func() {
		box0.Children = func() []*sumi.Input {
			var cs []*sumi.Input
			if connected.Get() {
				cs = append(cs, &sumi.Input{
					Kind:      sumi.KindBox,
					Padding:   sumi.ParsePadding("0 1"),
					CursorCol: -1,
					CursorRow: -1,
					Style: sumi.Style{
						FG:   sumi.Color{IsRGB: true, R: 80, G: 250, B: 123},
						Bold: true,
					},
					Children: []*sumi.Input{
						{
							Kind:    sumi.KindText,
							Content: "●",
						},
					},
				})
			} else {
				cs = append(cs, &sumi.Input{
					Kind:      sumi.KindBox,
					Padding:   sumi.ParsePadding("0 1"),
					CursorCol: -1,
					CursorRow: -1,
					Style: sumi.Style{
						FG:   sumi.Color{IsRGB: true, R: 255, G: 85, B: 85},
						Bold: true,
					},
					Children: []*sumi.Input{
						{
							Kind:    sumi.KindText,
							Content: "●",
						},
					},
				})
			}
			cs = append(cs, &sumi.Input{
				Kind:      sumi.KindBox,
				Direction: "row",
				CursorCol: -1,
				CursorRow: -1,
				Children: []*sumi.Input{
					{
						Kind:      sumi.KindBox,
						Padding:   sumi.ParsePadding("0 1"),
						CursorCol: -1,
						CursorRow: -1,
						Style: sumi.Style{
							FG:   sumi.Color{IsRGB: true, R: 139, G: 233, B: 253},
							Bold: true,
						},
						Children: []*sumi.Input{
							{
								Kind:    sumi.KindText,
								Content: "Console",
							},
						},
					},
					{
						Kind:      sumi.KindBox,
						Padding:   sumi.ParsePadding("0 1"),
						CursorCol: -1,
						CursorRow: -1,
						Style: sumi.Style{
							Dim: true,
						},
						Children: []*sumi.Input{
							{
								Kind:    sumi.KindText,
								Content: "Elements",
							},
						},
					},
					{
						Kind:      sumi.KindBox,
						Padding:   sumi.ParsePadding("0 1"),
						CursorCol: -1,
						CursorRow: -1,
						Style: sumi.Style{
							Dim: true,
						},
						Children: []*sumi.Input{
							{
								Kind:    sumi.KindText,
								Content: "Network",
							},
						},
					},
					{
						Kind:      sumi.KindBox,
						Padding:   sumi.ParsePadding("0 1"),
						CursorCol: -1,
						CursorRow: -1,
						Style: sumi.Style{
							Dim: true,
						},
						Children: []*sumi.Input{
							{
								Kind:    sumi.KindText,
								Content: "Sources",
							},
						},
					},
				},
			})
			return cs
		}()
	})

	return &sumi.Component{
		Tree: root,
	}
}
