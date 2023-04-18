// Package tui contains routines for creating the TUI.
package tui

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/util"
)

var (
	boxStyle          BoxStyle
	italicAsUnderline bool
	palettes          = []PaletteSpec{
		{false, 30, 37, 0},
		{true, 40, 47, 0},
		{false, 90, 97, 8},
		{true, 100, 107, 8},
	}
	// FIXME: this is ugly and feels out of place here
	Colors = struct {
		Comment,
		LineNumber,
		CurrentLineNumber,
		BoxOutline,
		Menu tcell.Style
	}{
		tcell.StyleDefault,
		tcell.StyleDefault,
		tcell.StyleDefault,
		tcell.StyleDefault,
		tcell.StyleDefault,
	}
	Alignment = struct{ Begin, Center, End, Fill int }{0, 1, 2, 3}
)

type PaletteSpec struct {
	isBackground  bool
	first, last   int
	paletteOffset int
}

type BoxStyle struct {
	Vertical       rune
	Horizontal     rune
	TopLeft        rune
	TopRight       rune
	BottomLeft     rune
	BottomRight    rune
	VerticalRight  rune
	VerticalLeft   rune
	HorizontalDown rune
	HorizontalUp   rune
	Cross          rune
}

func BoxStyleFromString(set string) BoxStyle {
	style := BoxStyle{}
	value := reflect.ValueOf(&style)
	fieldCount := reflect.ValueOf(style).NumField()
	runes := []rune(set)
	if fieldCount != len(runes) {
		panic(fmt.Sprintf(
			"BoxStyleFromString: set contains %d symbols, expected %d",
			len(runes),
			fieldCount),
		)
	}
	for i := 0; i < fieldCount; i++ {
		value.Elem().Field(i).SetInt(int64(runes[i]))
	}
	return style
}

func GetBoxStyle(description string) BoxStyle {
	switch description {
	default:
		log.Printf("unknown box style ‘%s’, using rounded", description)
		fallthrough
	case "rounded":
		return BoxStyleFromString("│─╭╮╰╯├┤┬┴┼")
	case "sharp":
		return BoxStyleFromString("│─┌┐└┘├┤┬┴┼")
	case "heavysharp":
		return BoxStyleFromString("┃━┏┓┗┛┣┫┳┻╋")
	case "double":
		return BoxStyleFromString("║═╔╗╚╝╠╣╦╩╬")
	case "ascii":
		return BoxStyleFromString("|-+++++++++")
	}
}

type Cell struct {
	char  rune
	style tcell.Style
}

func NewCell(char rune, style tcell.Style) Cell {
	return Cell{char, style}
}

func Init(cfg *Config) tcell.Screen {
	boxStyle = GetBoxStyle(cfg.General.BoxStyle)
	italicAsUnderline = cfg.General.ItalicToUnderline
	scr, err := tcell.NewScreen()
	if err != nil {
		Fatal("could not create screen: %s", err)
	}
	if err := scr.Init(); err != nil {
		Fatal("could not initialize screen: %s", err)
	}
	if cfg.General.Mouse {
		scr.EnableMouse(tcell.MouseMotionEvents)
	}
	Colors.Comment = Ansi2Style(cfg.Colors.Comment)
	Colors.LineNumber = Ansi2Style(cfg.Colors.LineNumber)
	Colors.CurrentLineNumber = Ansi2Style(cfg.Colors.CurrentLineNumber)
	Colors.BoxOutline = Ansi2Style(cfg.Colors.BoxOutline)
	Colors.Menu = Ansi2Style(cfg.Colors.Menu)
	return scr
}

func Quit(scr tcell.Screen) {
	scr.Fini()
}

func Text(scr tcell.Screen, x, y int, text string, style tcell.Style) int {
	for _, char := range text {
		scr.SetContent(x, y, char, nil, style)
		x += runewidth.RuneWidth(char)
	}
	return x
}

// TextWithHighlight prints the given strings, highlighting one character.
func TextWithHighlight(
	scr tcell.Screen,
	x, y int,
	text string,
	highlight int,
	normalStyle, highlightStyle tcell.Style,
) int {
	if highlight < 0 {
		return Text(scr, x, y, text, normalStyle)
	}
	var style tcell.Style
	for i, char := range text {
		if i == highlight {
			style = highlightStyle
		} else {
			style = normalStyle
		}
		scr.SetContent(x, y, char, nil, style)
		x += runewidth.RuneWidth(char)
	}
	return x
}

// RightText prints a right aligned string.
func RightText(scr tcell.Screen, x, y, width int, text string, style tcell.Style) {
	textWidth := runewidth.StringWidth(text)
	x += width - textWidth
	Text(scr, x, y, text, style)
}

func HLine(scr tcell.Screen, x, y int, width int, char rune, style tcell.Style) {
	for col := x; col < x+width; col++ {
		scr.SetContent(col, y, char, nil, style)
	}
}

func VLine(scr tcell.Screen, x, y int, height int, char rune, style tcell.Style) {
	for row := y; row < y+height; row++ {
		scr.SetContent(x, row, char, nil, style)
	}
}

func Box(scr tcell.Screen, x, y, width, height int, style tcell.Style) {
	right := x + width - 1
	bottom := y + height - 1
	scr.SetContent(x, y, boxStyle.TopLeft, nil, style)
	scr.SetContent(right, y, boxStyle.TopRight, nil, style)
	scr.SetContent(x, bottom, boxStyle.BottomLeft, nil, style)
	scr.SetContent(right, bottom, boxStyle.BottomRight, nil, style)
	HLine(scr, x+1, y, width-2, boxStyle.Horizontal, style)
	HLine(scr, x+1, bottom, width-2, boxStyle.Horizontal, style)
	VLine(scr, x, y+1, height-2, boxStyle.Vertical, style)
	VLine(scr, right, y+1, height-2, boxStyle.Vertical, style)
}

func FillRect(scr tcell.Screen, x, y, width, height int, char rune, style tcell.Style) {
	for row := y; row < y+height; row++ {
		HLine(scr, x, row, width, char, style)
	}
}

func getColor(tok int, tokens []int, style tcell.Style) (tcell.Style, []int) {
	isBackground := false
	color := tcell.ColorDefault
	for _, palette := range palettes {
		if tok >= palette.first && tok <= palette.last {
			isBackground = palette.isBackground
			color = tcell.PaletteColor(palette.paletteOffset + tok - palette.first)
			goto done
		}
	}
	if tok == 38 || tok == 48 {
		isBackground = tok == 48
		tok, tokens = util.PopFront(tokens)
		if tok == 5 {
			var index int
			index, tokens = util.PopFront(tokens)
			color = tcell.PaletteColor(index)
		} else if tok == 2 {
			var red, green, blue int
			red, tokens = util.PopFront(tokens)
			green, tokens = util.PopFront(tokens)
			blue, tokens = util.PopFront(tokens)
			color = tcell.NewRGBColor(int32(red), int32(green), int32(blue))
		}
		// silently ignore invalid color type
	} else if tok == 39 || tok == 49 {
		isBackground = tok == 49
		// color is already the default
	}
done:
	if isBackground {
		style = style.Background(color)
	} else {
		style = style.Foreground(color)
	}
	return style, tokens
}

// Ansi2Style converts a SGR ansi escape sequence to a tcell Style.
func Ansi2Style(sequence string) (style tcell.Style) {
	if len(sequence) == 0 {
		return tcell.StyleDefault
	}
	// Out lexer only understands sequences ending with 'm' so we don't need
	// to verify it's a SGR sequence.
	tokens := util.Map(
		strings.Split(sequence[2:len(sequence)-1], ";"),
		func(s string) int {
			i, _ := strconv.Atoi(s)
			// We can ignore any errors here, this will just give us 0 (reset)
			// for empty and invalid tokens which is the best way to handle
			// them anyways.
			return i
		},
	)
	var tok int
	for len(tokens) > 0 {
		tok, tokens = util.PopFront(tokens)
		switch tok {
		case 0:
			style = tcell.StyleDefault
		case 1:
			style = style.Bold(true)
		case 2:
			style = style.Dim(true)
		case 3:
			if italicAsUnderline {
				style = style.Underline(true)
			} else {
				style = style.Italic(true)
			}
		case 4:
			style = style.Underline(true)
		// blinking is left out
		case 7:
			style = style.Reverse(true)

		case 22:
			style = style.Bold(false).Dim(false)
		case 23:
			if italicAsUnderline {
				style = style.Underline(false)
			} else {
				style = style.Italic(false)
			}
		case 24:
			style = style.Underline(false)
		case 27:
			style = style.Reverse(false)

		case
			30, 31, 32, 33, 34, 35, 36, 37,
			38,
			39,
			40, 41, 42, 43, 44, 45, 46, 47,
			48,
			49:
			style, tokens = getColor(tok, tokens, style)
		}
	}
	return style
}

func alignAxis(avail, use, alignment int) (int, int) {
	switch alignment {
	case Alignment.Begin:
		return 0, use
	case Alignment.Center:
		return (avail - use) / 2, use
	case Alignment.End:
		return avail - use, use
	case Alignment.Fill:
		return 0, avail
	}
	panic("unreachable")
}

// TranslateControl gets the key and rune from the given event, translating
// upper and lowercase HJKL to arrow keys. Additionally converts space to enter.
func TranslateControls(ev *tcell.EventKey) (tcell.Key, rune) {
	if ev.Key() == tcell.KeyRune {
		switch ev.Rune() {
		case 'k', 'K':
			return tcell.KeyUp, 0
		case 'j', 'J':
			return tcell.KeyDown, 0
		case 'h', 'H':
			return tcell.KeyLeft, 0
		case 'l', 'L':
			return tcell.KeyRight, 0
		case ' ':
			return tcell.KeyEnter, 0
		default:
			return tcell.KeyRune, ev.Rune()
		}
	} else {
		return ev.Key(), ev.Rune()
	}
}
