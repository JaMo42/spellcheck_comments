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
	sf "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

const (
	appName    = "spellcheck_comments"
	appVersion = "0.1.0"
)

type Options struct {
	backup         bool
	applyBackup    bool
	applyBackupAll bool
	globs          []string
}

func parseArgs() (Options, []string) {
	InvocationName = os.Args[0]
	globsString := ""
	showVersion := false
	var options Options
	flag.StringVar(
		&globsString, "globs", "",
		"comma separated list of globs for file names in directories",
	)
	flag.BoolVar(
		&showVersion, "version", false,
		"show version information",
	)
	flag.BoolVar(
		&options.backup, "with-backup", false,
		"generate a backup even if disable in the config",
	)
	flag.BoolVar(
		&options.applyBackup, "apply-backup", false,
		"apply the backup, asking for each file",
	)
	var applyBackupAlias bool
	flag.BoolVar(&applyBackupAlias, "b", false, "alias for -apply-backup")
	flag.BoolVar(
		&options.applyBackupAll, "apply-backup-all", false,
		"apply the backup for all files",
	)
	var applyBackupAllAlias bool
	flag.BoolVar(&applyBackupAllAlias, "B", false, "alias for -apply-backup-all")
	flag.Parse()
	if showVersion {
		fmt.Printf("%s %s\n", appName, appVersion)
		os.Exit(0)
	}
	options.applyBackup = options.applyBackup || applyBackupAlias
	options.applyBackupAll = options.applyBackupAll || applyBackupAllAlias
	if len(globsString) != 0 {
		options.globs = util.Filter(
			strings.Split(globsString, ","),
			func(pattern string) bool {
				_, err := filepath.Match(pattern, "")
				if err != nil {
					log.Printf("%s: discarding invalid glob: %s\n", InvocationName, pattern)
				}
				return err == nil
			},
		)
	}
	return options, flag.Args()
}

func discover(files []string, dir string, filter func(string, bool) bool) []string {
	dirContent, _ := os.ReadDir(dir)
	for _, file := range dirContent {
		name := file.Name()
		if file.IsDir() {
			files = discover(files, fmt.Sprintf("%s/%s", dir, name), filter)
		} else if filter(name, false) {
			files = append(files, fmt.Sprintf("%s/%s", dir, name))
		}
	}
	return files
}

func getFiles(args []string, filter func(string, bool) bool) []string {
	files := []string{}
	if len(args) == 0 {
		return discover(files, ".", filter)
	} else {
		for _, arg := range args {
			stat, err := os.Stat(arg)
			if err != nil {
				log.Printf("%s: %s\n", InvocationName, err)
				continue
			}
			if stat.IsDir() {
				files = discover(files, arg, filter)
			} else if filter(arg, true) {
				files = append(files, arg)
			}
		}
	}
	return files
}

func fileFilter(cfg *Config, options *Options) func(filename string, direct bool) bool {
	extensionFilter := func(filename string, _ bool) bool {
		ext := fileExtension(filename)
		for _, extensions := range cfg.Extensions {
			if util.Contains(extensions, ext) {
				return true
			}
		}
		return false
	}
	if len(options.globs) == 0 {
		return extensionFilter
	}
	return func(filename string, direct bool) bool {
		if !extensionFilter(filename, direct) {
			return false
		}
		for _, glob := range options.globs {
			if match, _ := filepath.Match(glob, filename); match {
				return true
			}
		}
		if direct {
			log.Printf(
				"%s: skipping %s: no comment style defined for extension",
				InvocationName,
				filename,
			)
		}
		return false
	}
}

func noHighlight(filename string) (string, bool) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", true
	}
	return string(content), true
}

func highlight(filename string, cfg *Config) (string, bool) {
	if len(cfg.General.HighlightCommand) == 0 {
		return noHighlight(filename)
	}
	highlightCommand := strings.ReplaceAll(cfg.General.HighlightCommand, "%FILE%", filename)
	commandLine, err := shellquote.Split(highlightCommand)
	if err != nil {
		Fatal("syntax error in highlight command: %s", err)
	}
	stdoutData, err := exec.Command(commandLine[0], commandLine[1:]...).Output()
	if err != nil {
		return noHighlight(filename)
	}
	return string(stdoutData), false
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

func globalControls() []tui.KeyAction {
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

func parseFiles(names []string, cfg *Config, speller aspell.Speller, out chan sf.SourceFile) {
	for _, filename := range names {
		highlit, failed := highlight(filename, cfg)
		if len(highlit) == 0 {
			continue
		}
		style := cfg.GetStyle(fileExtension(filename))
		sf := parser.Parse(filename, highlit, style, speller, cfg, failed)
		if !sf.Ok() {
			out <- sf
		}
	}
	close(out)
}

func configPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if len(configHome) == 0 {
		home := os.Getenv("HOME")
		if len(home) == 0 {
			home = os.Getenv("Home")
		}
		if len(home) == 0 {
			return ""
		}
		configHome = fmt.Sprintf("%s/.config", home)
	}
	return filepath.Join(configHome, "spellcheck_comments.toml")
}

func main() {
	log.SetFlags(0)
	options, args := parseArgs()
	if options.applyBackup {
		RunBackup()
		return
	} else if options.applyBackupAll {
		BackupRestoreAll()
		return
	}
	var cfg Config
	if path := configPath(); len(path) != 0 {
		cfg = LoadConfig(path)
	} else {
		cfg = DefaultConfig()
	}

	files := getFiles(args, fileFilter(&cfg, &options))
	if len(files) == 0 {
		return
	}

	speller, err := aspell.NewSpeller(cfg.Aspell())
	if err != nil {
		Fatal("could not create speller: %s", err.Error())
	}
	defer speller.Delete()

	scr := tui.Init(&cfg)
	defer tui.Quit(scr)

	checker := NewSpellChecker(scr, speller, &cfg, &options)

	sourceFiles := make(chan sf.SourceFile)
	go parseFiles(files, &cfg, speller, sourceFiles)

	for sf := range sourceFiles {
		if checker.CheckFile(sf) {
			break
		}
	}
	checker.Finish()
}
