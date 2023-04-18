package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/term_input"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

const (
	iconNone    = " "
	iconWarn    = "\x1b[33m⚠\x1b[m"
	iconError   = "\x1b[31m❌\x1b[m"
	iconOk      = "\x1b[32m✓\x1b[m"
	iconSkipped = "\x1b[2m-\x1b[m"
)

func backupFileName() string {
	return fmt.Sprintf("%s.backup", appName)
}

func relPath(pathname string) string {
	cwd, _ := os.Getwd()
	rel, err := filepath.Rel(cwd, pathname)
	if err != nil {
		return pathname
	}
	return rel
}

type backupLine struct {
	isFile bool
	line   int
	time   int64
	text   string
}

type Backup struct {
	lines    []backupLine
	builder  strings.Builder
	lastLine int
	file     *os.File
}

func (self *Backup) SetFile(filename string) {
	pathname, _ := filepath.Abs(filepath.Clean(filename))
	stat, _ := os.Stat(pathname)
	self.lines = append(self.lines, backupLine{
		isFile: true,
		time:   stat.ModTime().Unix(),
		text:   pathname,
	})
	self.lastLine = -1
}

func (self *Backup) AddLine(line int, tb *tui.TextBuffer, replacements map[tui.SliceIndex]string) {
	if line == self.lastLine {
		return
	}
	self.builder.Reset()
	tb.ForEachInLine(line, func(s string, index tui.SliceIndex) {
		replacement, found := replacements[index]
		if found {
			self.builder.WriteString(replacement)
		} else {
			self.builder.WriteString(s)
		}
	})
	self.lines = append(self.lines, backupLine{
		line: line,
		text: self.builder.String(),
	})
	self.lastLine = line
}

// Create creates the backup file
func (self *Backup) Create() error {
	var err error
	self.file, err = os.Create(backupFileName())
	return err
}

// Write writes the backup file
func (self *Backup) Write() {
	w := bufio.NewWriter(self.file)
	for _, line := range self.lines {
		if line.isFile {
			fmt.Fprintf(w, "FILE %d %s\n", line.time, line.text)
		} else {
			fmt.Fprintf(w, "%d %s\n", line.line, line.text)
		}
	}
	w.Flush()
	self.file.Close()
}

// LoadBackup attempts to load a backup file from the current directory.
func LoadBackup() Optional[Backup] {
	data, err := os.ReadFile(backupFileName())
	if err != nil {
		return None[Backup]()
	}
	backup := Backup{}
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "FILE") {
			split := strings.SplitN(line, " ", 3)
			time, _ := strconv.ParseInt(split[1], 10, 64)
			pathname := split[2]
			backup.lines = append(backup.lines, backupLine{
				isFile: true,
				time:   time,
				text:   pathname,
			})
		} else {
			split := strings.SplitN(line, " ", 2)
			line, _ := strconv.ParseInt(split[0], 10, 32)
			text := split[1]
			backup.lines = append(backup.lines, backupLine{
				line: int(line),
				text: text,
			})
		}
	}
	return Some(backup)
}

// statAndCheck stats the given file and checks if it's a writable regular file.
func statAndCheck(pathname string) (os.FileInfo, error) {
	stat, err := os.Stat(pathname)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("files does not exist")
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	// FIXME: this assumes we are the user that created the file
	needPerm := fs.FileMode(0o600)
	if stat.Mode().Perm()&needPerm != needPerm {
		return nil, fmt.Errorf("file is not writable")
	}
	return stat, nil
}

type backupForEachCallback = func(pathname string, err error, outdated bool, lineNumbers []int, lines []string) bool

// ForEach calls the given function for each file. This consumes the backup data.
func (self *Backup) ForEach(f backupForEachCallback) {
	if !self.lines[0].isFile {
		Fatal("invalid backup: first line must be a file header")
	}
	var line backupLine
	lineNumbers := []int{}
	lines := []string{}
	for len(self.lines) != 0 {
		line, self.lines = util.PopFront(self.lines)
		pathname := line.text
		stat, err := statAndCheck(pathname)
		if err != nil {
			f(pathname, err, false, lineNumbers, lines)
			continue
		}
		outdated := stat.ModTime().Unix() != line.time
		lineNumbers = lineNumbers[:0]
		lines = lines[:0]
		for len(self.lines) != 0 && !self.lines[0].isFile {
			line, self.lines = util.PopFront(self.lines)
			lineNumbers = append(lineNumbers, line.line)
			lines = append(lines, line.text)
		}
		if !f(pathname, nil, outdated, lineNumbers, lines) {
			break
		}
	}
}

// backupApplyFile returns a callback for Backup.ForEach that applies the backup
// for one file with the specified error handling.
func backupApplyFile(pathname string, lineNumbers []int, lines []string) error {
	data, _ := os.ReadFile(pathname)
	fileLines := strings.Split(string(data), "\n")
	for i, line := range lineNumbers {
		fileLines[line] = lines[i]
	}
	file, _ := os.Create(pathname)
	w := bufio.NewWriter(file)
	for _, line := range fileLines {
		w.WriteString(line)
		w.WriteByte('\n')
	}
	err := w.Flush()
	file.Close()
	return err
}

func restoreAllForEach(pathname string, err error, outdated bool, lineNumbers []int, lines []string) bool {
	displayPath := relPath(pathname)
	if err != nil {
		log.Printf("%s: %s\n", displayPath, err)
	}
	if outdated {
		log.Printf("%s: file changed since backup (skipping)", displayPath)
	}
	err = backupApplyFile(pathname, lineNumbers, lines)
	if err != nil {
		log.Printf("%s: %s\n", displayPath, err)
	}
	return true
}

func BackupRestoreAll() {
	LoadBackup().Then(func(backup Backup) {
		backup.ForEach(restoreAllForEach)
	}).Else(func() {
		fmt.Println("No backup file in current directory")
	})
}

func runBackupForEach(pathname string, err error, outdated bool, lineNumbers []int, lines []string) bool {
	displayPath := relPath(pathname)
	var textLen int
	if err != nil {
		fmt.Printf("%s %s: %s\n", iconError, displayPath, err)
		return true
	} else if outdated {
		textLen, _ = fmt.Printf("%s %s (file changed since backup!) (y/N)", iconWarn, displayPath)
	} else {
		textLen, _ = fmt.Printf("%s %s (Y/n)", iconNone, displayPath)
	}
	defer fmt.Println("\r")
	yes := false
inputLoop:
	for {
		switch term_input.Read() {
		case 'y', 'Y':
			yes = true
			break inputLoop
		case 'n', 'N':
			break inputLoop
		case '\r', '\n':
			if !outdated {
				yes = true
			}
			break inputLoop
		case 0:
			return false
		}
	}
	if yes {
		err = backupApplyFile(pathname, lineNumbers, lines)
		if err != nil {
			clearOld := strings.Repeat(" ", textLen+1)
			fmt.Printf("\r%s\r%s %s: %s", clearOld, iconError, displayPath, err)
		} else {
			fmt.Printf("\r%s", iconOk)
		}
	} else {
		fmt.Printf("\r%s", iconSkipped)
	}
	return true
}

func RunBackup() {
	LoadBackup().Then(func(backup Backup) {
		fd := int(os.Stdin.Fd())
		oldState, _ := term.MakeRaw(fd)
		term_input.Begin()
		backup.ForEach(runBackupForEach)
		term_input.Stop()
		term.Restore(fd, oldState)
	}).Else(func() {
		fmt.Println("No backup file in current directory")
	})
}
