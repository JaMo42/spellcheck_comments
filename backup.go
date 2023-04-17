package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JaMo42/spellcheck_comments/tui"
)

type BackupLine struct {
	isFile bool
	line   int
	time   int64
	text   string
}

type Backup struct {
	lines    []BackupLine
	builder  strings.Builder
	lastLine int
	file     *os.File
}

func (self *Backup) SetFile(filename string) {
	pathname, _ := filepath.Abs(filepath.Clean(filename))
	stat, _ := os.Stat(pathname)
	self.lines = append(self.lines, BackupLine{
		isFile: true,
		time:   stat.ModTime().Unix(),
		text:   pathname,
	})
	self.lastLine = -1
}

func (self *Backup) AddLine(line int, tb *tui.TextBuffer) {
	if line == self.lastLine {
		return
	}
	self.builder.Reset()
	tb.ForEachInLine(line, func(s string) {
		self.builder.WriteString(s)
	})
	self.lines = append(self.lines, BackupLine{
		line: line,
		text: self.builder.String(),
	})
	self.lastLine = line
}

func (self *Backup) Create() error {
	var err error
	self.file, err = os.Create(fmt.Sprintf("%s.backup", appName))
	return err
}

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
