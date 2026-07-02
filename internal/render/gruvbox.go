package render

import (
	"charm.land/glamour/v2/ansi"
)

// Gruvbox Dark Hard palette (morhetz/gruvbox), matching the app theme.
const (
	gruvboxBg0Hard = "#1d2021"
	gruvboxBg1     = "#3c3836"
	gruvboxBg3     = "#665c54"
	gruvboxFg1     = "#ebdbb2"
	gruvboxFg3     = "#bdae93"
	gruvboxGray    = "#928374"
	gruvboxRed     = "#fb4934"
	gruvboxGreen   = "#b8bb26"
	gruvboxYellow  = "#fabd2f"
	gruvboxBlue    = "#83a598"
	gruvboxPurple  = "#d3869b"
	gruvboxAqua    = "#8ec07c"
	gruvboxOrange  = "#fe8019"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func uintPtr(u uint) *uint    { return &u }

// GruvboxDarkHard returns a Glamour style configured with the Gruvbox Dark
// Hard palette so Markdown rendering matches the rest of the Trainer UI.
func GruvboxDarkHard() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       strPtr(gruvboxFg1),
			},
			Margin: uintPtr(2),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  strPtr(gruvboxFg3),
				Italic: boolPtr(true),
			},
			Indent: uintPtr(1),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
		},
		List: ansi.StyleList{
			LevelIndent: 2,
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: strPtr(gruvboxFg1),
				},
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       strPtr(gruvboxYellow),
				Bold:        boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           strPtr(gruvboxBg0Hard),
				BackgroundColor: strPtr(gruvboxYellow),
				Bold:            boolPtr(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "## ",
				Color:  strPtr(gruvboxYellow),
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "### ",
				Color:  strPtr(gruvboxAqua),
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "#### ",
				Color:  strPtr(gruvboxAqua),
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "##### ",
				Color:  strPtr(gruvboxAqua),
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "###### ",
				Color:  strPtr(gruvboxAqua),
				Bold:   boolPtr(false),
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Color:  strPtr(gruvboxFg3),
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolPtr(true),
			Color: strPtr(gruvboxOrange),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  strPtr(gruvboxBg3),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
			Color:       strPtr(gruvboxBlue),
		},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{},
			Ticked:         "[✓] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     strPtr(gruvboxBlue),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: strPtr(gruvboxPurple),
			Bold:  boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color:     strPtr(gruvboxBlue),
			Underline: boolPtr(true),
		},
		ImageText: ansi.StylePrimitive{
			Color:  strPtr(gruvboxGray),
			Format: "Image: {{.text}} →",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           strPtr(gruvboxGreen),
				BackgroundColor: strPtr(gruvboxBg1),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: strPtr(gruvboxFg1),
				},
				Margin: uintPtr(2),
			},
			Theme: "gruvbox",
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{},
			},
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n🠶 ",
		},
	}
}
