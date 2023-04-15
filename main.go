package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kballard/go-shellquote"
	"github.com/trustmaster/go-aspell"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/parser"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

const (
	appName    = "spellcheck_comments"
	appVersion = "0.1.0"
)

var (
	globs []string
)

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
	ext := fileExtension(filename)
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
	return *util.Back(strings.Split(filename, "."))
}

func waitForAnyKey(scr tcell.Screen) bool {
	for {
		ev := scr.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			return ev.Key() == tcell.KeyCtrlC
		}
	}
}

type GlobalControl struct {
	key    rune
	label  string
	action any
}

func (self *GlobalControl) Key() rune {
	return self.key
}

func (self *GlobalControl) Label() string {
	return self.label
}

func (self *GlobalControl) Action() any {
	return self.action
}

func GlobalControls() []tui.KeyAction {
	x := func(k rune, l string, a any) *GlobalControl {
		control := new(GlobalControl)
		*control = GlobalControl{k, l, a}
		return control
	}
	return []tui.KeyAction{
		x('i', "Ignore", ActionIgnore{false}),
		x('I', "Ignore all", ActionIgnore{true}),
		x('x', "Exit", ActionExit{}),
		x('b', "Abort", ActionAbort{}),
	}
}

func main() {
	log.SetFlags(0)
	args := parseArgs()
	cfg := LoadConfig("config.toml")
	files := util.Filter(
		getFiles(args),
		func(filename string) bool {
			return careAboutFile(filename, &cfg)
		},
	)
	UNUSED(files)
	files = []string{"source_files/timer.hpp"}
	//files = []string{"source_files/hello_world.c"}

	speller, err := aspell.NewSpeller(cfg.Aspell())
	if err != nil {
		Fatal("could not create speller: %s", err.Error())
	}
	defer speller.Delete()

	scr := tui.Init(&cfg)
	defer tui.Quit(scr)

	checker := NewSpellChecker(scr, speller, &cfg)

	for _, filename := range files {
		fmt.Println(filename)
		content := highlight(filename, &cfg)
		style := cfg.GetStyle(fileExtension(filename))
		sf := parser.Parse(filename, content, style, speller, cfg.General.DimCode)
		if sf.Ok() {
			continue
		}
		checker.CheckFile(&sf)
	}
}
