package common

import (
	"errors"
	"os"

	"github.com/pelletier/go-toml/v2"
)

const (
	commentColorDefault = "\000DEFAULT"
)

type CfgGeneral struct {
	OnlyIfSpaceAfter  bool   `toml:"only-if-space-after"`
	HighlightCommand  string `toml:"highlight-command"`
	DimCode           bool   `toml:"dim-code"`
	BoxStyle          string `toml:"box-style"`
	ItalicToUnderline bool   `toml:"italic-to-underline"`
}

type CfgColors struct {
	Comment           string `toml:"comment-color"`
	LineNumber        string `toml:"line-number"`
	CurrentLineNumber string `toml:"current-line-number"`
	BoxOutline        string `toml:"box-outline"`
	MenuBackground    string `toml:"menu-background"`
	MenuText          string `toml:"menu-text"`
}

type Config struct {
	Extensions map[string][]string
	Styles     map[string]CommentStyle
	General    CfgGeneral
	Colors     CfgColors
}

func DefaultConfig() Config {
	return Config{
		General: CfgGeneral{
			OnlyIfSpaceAfter:  true,
			HighlightCommand:  "",
			DimCode:           true,
			BoxStyle:          "rounded",
			ItalicToUnderline: false,
		},
		Colors: CfgColors{
			Comment:           commentColorDefault,
			LineNumber:        "\x1b[38;5;243m",
			CurrentLineNumber: "\x1b[38;5;251m",
			BoxOutline:        "\x1b[38;5;213m",
			MenuBackground:    "\x1b[48;5;61m",
			MenuText:          "\x1b[48;5;232m",
		},
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
		if len(cfg.General.HighlightCommand) == 0 {
			cfg.Colors.Comment = "\x1b[1;32m"
		} else {
			cfg.Colors.Comment = ""
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
