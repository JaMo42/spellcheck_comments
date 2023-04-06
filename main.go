package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kballard/go-shellquote"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/parser"
	"github.com/JaMo42/spellcheck_comments/util"
)

const (
	appName    = "spellcheck_comments"
	appVersion = "0.1.0"
)

var (
	globs []string
)

type SourceLocation struct {
	Line, Column int
}

type Line struct {
	CommentText string
	Location    SourceLocation
}

type SourceFile []Line

type Project map[string]SourceFile

func UNUSED(x ...interface{}) {}

func parseArgs() []string {
	InvocationName = os.Args[0]
	globsString := "*"
	showVersion := false
	flag.StringVar(
		&globsString, "globs", "*.*",
		"comma separated list of globs for file names in directories",
	)
	flag.BoolVar(
		&showVersion, "version", false,
		"show version information",
	)
	flag.Parse()
	if showVersion {
		fmt.Printf("%s %s\n", appName, appVersion)
		os.Exit(0)
	}
	globs = util.Filter(
		strings.Split(globsString, ","),
		func(pattern string) bool {
			_, err := filepath.Match(pattern, "")
			if err != nil {
				log.Printf("%s: discarding invalid glob: %s\n", InvocationName, pattern)
			}
			return err == nil
		},
	)
	return flag.Args()
}

func discover(files []string, dir string) []string {
	dirContent, _ := os.ReadDir(dir)
	for _, file := range dirContent {
		name := file.Name()
		if file.IsDir() {
			files = discover(files, fmt.Sprintf("%s/%s", dir, name))
		} else {
			for _, glob := range globs {
				if match, _ := filepath.Match(glob, name); match {
					files = append(files, fmt.Sprintf("%s/%s", dir, name))
					break
				}
			}
		}
	}
	return files
}

func getFiles(args []string) []string {
	files := make([]string, len(args))
	if len(args) == 0 {
		return discover(files, ".")
	} else {
		for _, arg := range args {
			file, err := os.Open(arg)
			if err != nil {
				log.Printf("%s: %s\n", InvocationName, err)
				continue
			}
			stat, _ := file.Stat()
			if stat.IsDir() {
				files = discover(files, arg)
			} else {
				files = append(files, arg)
			}
		}
	}
	return files
}

func careAboutFile(filename string, cfg *Config) bool {
	split := strings.Split(filename, ".")
	ext := split[len(split)-1]
	for _, extensions := range cfg.Extensions {
		if util.Contains(extensions, ext) {
			return true
		}
	}
	return false
}

func highlight(filename string, cfg *Config) string {
	highlightCommand := strings.ReplaceAll(cfg.General.HighlightCommand, "%FILE%", filename)
	commandLine, err := shellquote.Split(highlightCommand)
	if err != nil {
		Fatal("syntax error in highlight command: %s", err)
	}
	stdoutData, err := exec.Command(commandLine[0], commandLine[1:]...).Output()
	if err != nil {
		// TODO: no need to fatal here, just continue without highlighting
		Fatal("highlight command failed: %s", err)
	}
	return string(stdoutData)
}

func fileExtension(filename string) string {
	split := strings.Split(filename, ".")
	return split[len(split)-1]
}

func main() {
	log.SetFlags(0)
	args := parseArgs()
	cfg := LoadConfig("config.toml")
	// unknown extensions aren't an error so we silently filter here
	files := util.Filter(
		getFiles(args),
		func(filename string) bool {
			return careAboutFile(filename, &cfg)
		},
	)
	UNUSED(files)
	//files = []string{"source_files/timer.hpp"}
	files = []string{"source_files/hello_world.c"}

	for _, filename := range files {
		fmt.Println(filename)
		content := highlight(filename, &cfg)
		style := cfg.GetStyle(fileExtension(filename))
		lexer := parser.NewLexer(content, style)
		color := "\x1b[0m"
		for {
			tok := lexer.Next()
			fmt.Printf("%s('%s')\n", parser.LexerTokenKindName(tok.Kind()), strings.ReplaceAll(tok.Text(), "\x1b", "\\e"))
			if tok.Kind() == parser.TokenKind.EOF {
				break
			}
			continue
			if tok.Kind() == parser.TokenKind.EOF {
				break
			} else if tok.Kind() == parser.TokenKind.Newline {
				fmt.Printf("\x1b[0;2m\\n%s\n", color)
			} else {
				if tok.Kind() == parser.TokenKind.Style {
					color = tok.Text()
					//fmt.Print("\x1b[0;2mc\x1b[22m")
				}
				if tok.Kind() == parser.TokenKind.CommentWord {
					fmt.Printf("\x1b[0;2m{\x1b[m%s%s\x1b[0;2m}\x1b[m%s", color, tok.Text(), color)
				} else {
					fmt.Print(tok.Text())
				}
			}
		}
	}
}
