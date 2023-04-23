package common

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	commentColorDefault = "\000DEFAULT"
	DefaultCommentColor = "\x1b[1;32m"
)

var FallbackCommentColor string

type CfgGeneral struct {
	Backup              bool     `toml:"backup"`
	BottomStatus        bool     `toml:"bottom-status"`
	BoxStyle            string   `toml:"box-style"`
	DimCode             bool     `toml:"dim-code"`
	FilterCommentedCode bool     `toml:"filter-commented-code"`
	Filters             []string `toml:"filters"`
	HighlightCommands   []string `toml:"highlight-commands"`
	IgnoreCase          bool     `toml:"ignore-case"`
	IgnoreLists         []string `toml:"ignore-lists"`
	ItalicToUnderline   bool     `toml:"italic-to-underline"`
	Layout              string   `toml:"layout"`
	Mouse               bool     `toml:"mouse"`
	Suggestions         int      `toml:"suggestions"`
	TabSize             int      `toml:"tab-size"`
}

type CfgColors struct {
	BoxOutline        string `toml:"box-outline"`
	Comment           string `toml:"comment-color"`
	CurrentLineNumber string `toml:"current-line-number"`
	LineNumber        string `toml:"line-number"`
	Menu              string `toml:"menu"`
	StatusBar         string `toml:"status-bar"`
}

type Config struct {
	Extensions    map[string][]string
	Styles        map[string]CommentStyle
	General       CfgGeneral
	Colors        CfgColors
	AspellOptions map[string]string `toml:"aspell-options"`
}

func DefaultConfig() Config {
	return Config{
		Extensions: make(map[string][]string),
		Styles:     make(map[string]CommentStyle),
		General: CfgGeneral{
			Backup:              true,
			BottomStatus:        false,
			BoxStyle:            "rounded",
			DimCode:             true,
			FilterCommentedCode: false,
			Filters:             []string{},
			HighlightCommands:   []string{},
			IgnoreCase:          true,
			IgnoreLists:         []string{".spellcheck_comments_ignorelist"},
			ItalicToUnderline:   false,
			Layout:              "default",
			Mouse:               true,
			Suggestions:         -1,
			TabSize:             4,
		},
		Colors: CfgColors{
			BoxOutline:        "\x1b[38;5;213m",
			Comment:           commentColorDefault,
			CurrentLineNumber: "\x1b[38;5;251m",
			LineNumber:        "\x1b[38;5;243m",
			Menu:              "\x1b[48;5;61;38;5;232m",
			StatusBar:         "\x1b[38;5;251;7m",
		},
		AspellOptions: make(map[string]string),
	}
}

func LoadConfig(pathname string) Config {
	data, _ := os.ReadFile(pathname)
	cfg := DefaultConfig()
	err := toml.Unmarshal(data, &cfg)
	var derr *toml.DecodeError
	if errors.As(err, &derr) {
		Fatal("%v:\n%s", err, derr.String())
	}
	for name, style := range cfg.Styles {
		if err := style.Check(); err != nil {
			Fatal("invalid comment style: %s: %s", name, err)
		}
	}
	if cfg.Colors.Comment == commentColorDefault {
		if len(cfg.General.HighlightCommands) == 0 {
			cfg.Colors.Comment = DefaultCommentColor
		} else {
			cfg.Colors.Comment = ""
		}
		FallbackCommentColor = DefaultCommentColor
	} else {
		FallbackCommentColor = cfg.Colors.Comment
	}
	if cfg.General.Suggestions < 0 {
		if cfg.General.Layout == "aspell" {
			cfg.General.Suggestions = 10
		} else {
			cfg.General.Suggestions = 20
		}
	}
	return cfg
}

func (self *Config) GetStyle(extension string) CommentStyle {
	for style, extensions := range self.Extensions {
		for _, ext := range extensions {
			if ext == extension {
				return self.Styles[style]
			}
		}
	}
	panic("unreachable")
}

func (self *Config) Aspell() map[string]string {
	return self.AspellOptions
}

func (self *Config) DumpStyles() {
	type Style struct {
		name  string
		style CommentStyle
	}
	styles := []Style{}
	for name, style := range self.Styles {
		styles = append(styles, Style{name, style})
	}
	sort.Slice(styles, func(i, j int) bool {
		a := styles[i].name
		b := styles[j].name
		aBuiltin := strings.HasPrefix(a, "builtin")
		bBuiltin := strings.HasPrefix(b, "builtin")
		if aBuiltin && !bBuiltin {
			return true
		} else if bBuiltin && !aBuiltin {
			return false
		}
		if aBuiltin {
			a = a[8:]
			b = b[8:]
		}
		return a < b
	})
	first := true
	for _, pair := range styles {
		if !first {
			fmt.Println()
		}
		first = false
		pair.style.Dump(pair.name, self.Extensions[pair.name])
	}
}
