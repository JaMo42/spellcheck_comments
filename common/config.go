package common

import (
	"errors"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type General struct {
	OnlyIfSpaceAfter  bool   `toml:"only-if-space-after"`
	HighlightCommand  string `toml:"highlight-command"`
	DimCode           bool   `toml:"dim-code"`
	CommentColor      string `toml:"comment-color"`
	BoxStyle          string `toml:"box-style"`
	ItalicToUnderline bool   `toml:"italic-to-underline"`
}

type Config struct {
	Extensions map[string][]string
	Styles     map[string]CommentStyle
	General    General
}

func DefaultConfig() Config {
	return Config{
		General: General{
			OnlyIfSpaceAfter:  true,
			HighlightCommand:  "",
			DimCode:           true,
			CommentColor:      "",
			BoxStyle:          "rounded",
			ItalicToUnderline: false,
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
	if len(cfg.General.HighlightCommand) == 0 {
		cfg.General.CommentColor = "\x1b[1;32m"
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
