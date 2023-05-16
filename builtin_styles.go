package main

import (
	"github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/util"
	"golang.org/x/text/cases"
)

type styleData struct {
	name        string
	extenstions []string
	style       common.CommentStyle
}

// Note: there is no special support for character literals.
var defaultStringStyle = []common.StringStyle{
	{Begin: "\"", End: "\"", Escape: "\\\""},
	{Begin: "'", End: "'", Escape: "\\'"},
}

// Source: https://en.wikipedia.org/wiki/Comparison_of_programming_languages_(syntax)#Comments
var builtinStyles = []styleData{
	{
		name: "builtin-c",
		extenstions: []string{
			"c", "cc", "cpp", "cxx", "h", "hh", "hpp", "hxx",
			"go",
			"js",
			"cs",
			"java",
		},
		style: common.CommentStyle{
			Line:       []string{"//"},
			BlockBegin: []string{"/*"},
			BlockEnd:   []string{"*/"},
			Strings:    defaultStringStyle,
		},
	},
	{
		name:        "builtin-rust",
		extenstions: []string{"rs"},
		style: common.CommentStyle{
			Line:         []string{"//"},
			BlockBegin:   []string{"/*"},
			BlockEnd:     []string{"*/"},
			BlockNesting: true,
			// We cannot match character literals as the ' also appears in lifetimes
			Strings: []common.StringStyle{
				// Add a special case for '"' so we don't match the string begin
				// inside that. This must be before the other kind as their ends
				// overlap.
				{Begin: "'\"", End: "'"},
				{Begin: "\"", End: "\"", Escape: "\\\""},
				{Begin: "r#\"", End: "\"#"},
			},
		},
	},
	{
		name:        "builtin-python",
		extenstions: []string{"py"},
		style: common.CommentStyle{
			Line:       []string{"#"},
			BlockBegin: []string{"\"\"\"", "'''"},
			BlockEnd:   []string{"\"\"\"", "'''"},
			// We cannot support strings with the current parser as we'd always
			// match the strings over the doc strings.
			// Luckily we can only wrongly identify line comments so we don't
			// need to worry about unbalanced block comment tokens.
		},
	},
	{
		name:        "builtin-#",
		extenstions: []string{"sh", "bashrc", "toml", "ini", "cfg", "rb"},
		style: common.CommentStyle{
			Line:    []string{"#"},
			Strings: defaultStringStyle,
		},
	},
}

// MergeBuiltinStyles merges the builtin styles into the given config.
// Extensions that are already set are removed, if a builtin style has no
// unset extensions it is skipped.
func MergeBuiltinStyles(cfg *common.Config) {
	set := map[string]bool{}
	caser := cases.Fold()
	for _, extensions := range cfg.Extensions {
		for _, ext := range extensions {
			set[caser.String(ext)] = true
		}
	}
	for _, style := range builtinStyles {
		// Additional extensions for builtin types may be specified in the config.
		predef := cfg.Extensions[style.name]
		extensions := util.Filter(style.extenstions, func(ext string) bool {
			return !set[ext]
		})
		// We don't need to de-duplicate as predefined extensions are in the set
		// and got filtered from our extensions.
		extensions = append(extensions, predef...)
		if len(extensions) == 0 {
			continue
		}
		cfg.Styles[style.name] = style.style
		cfg.Extensions[style.name] = extensions
	}
}
