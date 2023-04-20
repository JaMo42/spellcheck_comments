package main

import (
	"bufio"
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
	appVersion = "0.2.0"
)

type Options struct {
	backup         bool
	applyBackup    bool
	applyBackupAll bool
	globs          []string
	dumpStyles     bool
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
	flag.BoolVar(
		&options.dumpStyles, "dump-styles", false,
		"Dump all configured styles to standard output.",
	)
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

// discover walks a directory tree, adding all files matching the filter to the
// files list. filter is the same as in getFiles.
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

// getFiles gets the list of files based on the arguments. If an argument
// specifies a file it is added to the list if it matches the filter.
// If it specified a directory it is recursively traversed, adding all files
// matching the filter. The filter receives the name if the file and whether it
// was an argument or found during directory discovery.
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

// fileFilter returns a filter for use in the getFiles function. The returned
// filter checks if  a comment style is defined for the files extension and
// whether it matches the glob filter option, if provided.
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

// noHighlight is the error case for the highlight function.
func noHighlight(filename string) (string, bool) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", true
	}
	return string(content), true
}

// highlight attempts to highlight a file using the highlighters defined in the
// config. If no highlighter is capable of highlighting the file the raw
// contents are returned. If the file cannot be read for any reason an empty
// string is returned. The returned bool is true if highlighting failed.
func highlight(filename string, cfg *Config) (string, bool) {
	if len(cfg.General.HighlightCommands) == 0 {
		return noHighlight(filename)
	}
	var stdoutData []byte
	for _, cmd := range cfg.General.HighlightCommands {
		if len(cmd) == 0 {
			continue
		}
		highlightCommand := strings.ReplaceAll(cmd, "%FILE%", filename)
		commandLine, err := shellquote.Split(highlightCommand)
		if err != nil {
			Fatal("syntax error in highlight command: %s", err)
		}
		stdoutData, err = exec.Command(commandLine[0], commandLine[1:]...).Output()
		if err == nil {
			break
		}
	}
	if len(stdoutData) == 0 {
		return noHighlight(filename)
	}
	return string(stdoutData), false
}

func fileExtension(filename string) string {
	return *util.Back(strings.Split(filename, "."))
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
		x('r', "Replace", ActionReplace{false}),
		x('R', "Replace all", ActionReplace{true}),
		x('u', "Undo last change", ActionUndo{}),
		x('s', "Skip rest of file", ActionSkip{}),
		x('x', "Exit", ActionExit{}),
		x('b', "Abort", ActionAbort{}),
	}
}

func parseFiles(
	names []string,
	cfg *Config,
	speller aspell.Speller,
	ignoreList *IgnoreList,
	out chan sf.SourceFile,
) {
	for _, filename := range names {
		highlit, failed := highlight(filename, cfg)
		if len(highlit) == 0 {
			continue
		}
		style := cfg.GetStyle(fileExtension(filename))
		sf := parser.Parse(
			filename,
			highlit,
			style,
			speller,
			cfg,
			ignoreList,
			failed,
		)
		if !sf.Ok() {
			out <- sf
		}
	}
	close(out)
}

type Paths struct {
	ConfigFile string
	ConfigDir  Optional[string]
}

// configPath returns the path and directory of the config file.
func configPath() (Paths, bool) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if len(configHome) == 0 {
		home := os.Getenv("HOME")
		if len(home) == 0 {
			home = os.Getenv("Home")
		}
		if len(home) == 0 {
			return Paths{}, false
		}
		configHome = fmt.Sprintf("%s/.config", home)
	}
	locations := []struct{ dir, file string }{
		{
			configHome,
			fmt.Sprintf("%s/spellcheck_comments.toml", configHome),
		},
		{
			fmt.Sprintf("%s/spellcheck_comments", configHome),
			fmt.Sprintf("%s/spellcheck_comments/config.toml", configHome),
		},
	}
	for _, location := range locations {
		stat, err := os.Stat(location.file)
		if err == nil && !stat.IsDir() {
			return Paths{location.file, Some(location.dir)}, true
		}
	}
	return Paths{}, false
}

// collectIgnoreLists creates the ignore list from all ignore list fiels to use.
func collectIgnoreLists(configPath Optional[string], cfg *Config) IgnoreList {
	dirs := []string{}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}
	configPath.Then(func(path string) {
		dirs = append(dirs, path)
	})
	list := NewIgnoreList(cfg.General.IgnoreCase)
	list.Add("todo")
	list.Add("fixme")
	for _, filename := range cfg.General.IgnoreLists {
		for _, dir := range dirs {
			pathname := fmt.Sprintf("%s/%s", dir, filename)
			file, err := os.Open(pathname)
			if err != nil {
				continue
			}
			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				word := scanner.Text()
				if len(word) > 1 {
					list.Add(word)
				}
			}
		}
	}
	return list
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
	paths, haveConfig := configPath()
	var cfg Config
	if haveConfig {
		cfg = LoadConfig(paths.ConfigFile)
	} else {
		cfg = DefaultConfig()
	}
	MergeBuiltinStyles(&cfg)
	if options.dumpStyles {
		cfg.DumpStyles()
		return
	}

	ignoreList := collectIgnoreLists(paths.ConfigDir, &cfg)

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
	tui.Text(scr, 0, 0, "Waiting for highlighter", tcell.StyleDefault)
	scr.Show()

	checker := NewSpellChecker(scr, speller, &cfg, &options)

	sourceFiles := make(chan sf.SourceFile)
	go parseFiles(files, &cfg, speller, &ignoreList, sourceFiles)

	allOk := true
	for sf := range sourceFiles {
		allOk = false
		checker.AddFile(sf)
		if checker.Run() {
			break
		}
	}
	checker.Finish()
	if allOk {
		scr.Suspend()
		fmt.Println("All files OK")
	}
}
