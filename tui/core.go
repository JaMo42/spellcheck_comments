package tui

import (
	"log"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/util"
)

type BoxStyle struct {
	vertical    rune
	horizontal  rune
	topLeft     rune
	topRight    rune
	bottomLeft  rune
	bottomRight rune
}

func GetBoxStyle(description string) BoxStyle {
	switch description {
	default:
		log.Printf("unknown box style '%s', using rounded", description)
		fallthrough
	case "rounded":
		return BoxStyle{'─', '│', '╭', '╮', '╰', '╯'}
	case "sharp":
		return BoxStyle{'─', '│', '┌', '┐', '└', '┘'}
	case "heavysharp":
		return BoxStyle{'━', '┃', '┏', '┓', '┗', '┛'}
	case "double":
		return BoxStyle{'═', '║', '╔', '╗', '╚', '╝'}
	case "ascii":
		return BoxStyle{'-', '|', '+', '+', '+', '+'}
	}
}

type PaletteSpec struct {
	isBackground  bool
	first, last   int
	paletteOffset int
}

var (
	boxStyle          BoxStyle
	italicAsUnderline bool
	palettes          = []PaletteSpec{
		{false, 30, 37, 0},
		{true, 40, 47, 0},
		{false, 90, 97, 8},
		{true, 100, 107, 8},
	}
)

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
	return scr
}

func Quit(scr tcell.Screen) {
	scr.Fini()
}

func Text(scr tcell.Screen, x, y int, text string, style tcell.Style) {
	for _, char := range text {
		scr.SetContent(x, y, char, nil, style)
		x += 1
	}
}

func HLine(scr tcell.Screen, x, y int, width int, char rune, style tcell.Style) {
	for col := x; col < x+width; col += 1 {
		scr.SetContent(col, y, char, nil, style)
	}
}

func VLine(scr tcell.Screen, x, y int, height int, char rune, style tcell.Style) {
	for row := x; row < y+height; row += 1 {
		scr.SetContent(x, row, char, nil, style)
	}
}

func Box(scr tcell.Screen, x, y, width, height int, style tcell.Style) {
	right := x + width - 1
	bottom := y + height - 1
	scr.SetContent(x, y, boxStyle.topLeft, nil, style)
	scr.SetContent(right, y, boxStyle.topRight, nil, style)
	scr.SetContent(x, bottom, boxStyle.bottomLeft, nil, style)
	scr.SetContent(right, bottom, boxStyle.bottomRight, nil, style)
	HLine(scr, x+1, y, width-2, boxStyle.horizontal, style)
	HLine(scr, x+1, bottom, width-2, boxStyle.horizontal, style)
	VLine(scr, x, y+1, height-2, boxStyle.vertical, style)
	VLine(scr, right, y+1, height-2, boxStyle.vertical, style)
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
		tok, tokens = util.Xxs(tokens)
		if tok == 2 {
			var index int
			index, tokens = util.Xxs(tokens)
			color = tcell.PaletteColor(index)
		} else if tok == 5 {
			var red, green, blue int
			red, tokens = util.Xxs(tokens)
			green, tokens = util.Xxs(tokens)
			blue, tokens = util.Xxs(tokens)
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
		tok, tokens = util.Xxs(tokens)
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
