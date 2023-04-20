package main

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/trustmaster/go-aspell"
	"golang.org/x/text/cases"

	. "github.com/JaMo42/spellcheck_comments/common"
	. "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

type ActionSelectSuggestion struct{ index int }
type ActionIgnore struct{ all bool }
type ActionReplace struct{ all bool }
type ActionUndo struct{}
type ActionSkip struct{}
type ActionExit struct{}
type ActionAbort struct{}

type Layout interface {
	SetSource(*SourceFile)
	Show(tui.SliceIndex)
	SetSuggestions([]string)
	ArrowReceiver() tui.ArrowReceiver
	MouseReceivers() []tui.MouseReceiver
}

type UndoReplacement struct {
	slice tui.SliceIndex
}

type UndoIgnore struct {
	all  bool
	word string
}

type UndoSkip struct{}

type UndoReplaceAll struct {
	startIndex tui.SliceIndex
	from       string
}

type UndoEventBase struct {
	fileId int
	wordId int
	kind   any
}

type SpellChecker struct {
	scr          tcell.Screen
	ui           tui.Tui
	layout       Layout
	speller      aspell.Speller
	ignore       map[string]bool
	replacements map[string]string
	changed      bool
	discardAll   bool
	doBackup     bool
	files        []FileContext
	caser        *cases.Caser
	currentFile  int
	undoStack    []UndoEventBase
}

func NewSpellChecker(
	scr tcell.Screen, speller aspell.Speller, cfg *Config, options *Options,
) SpellChecker {
	var layout Layout
	switch cfg.General.Layout {
	case "aspell":
		layout = new(AspellLayout)
	default:
		layout = new(DefaultLayout)
	}
	ui := tui.NewTui(scr, layout.(tui.Layout))
	ui.SetArrowReceiver(layout.ArrowReceiver())
	ui.SetMouseReceivers(layout.MouseReceivers())
	ui.SetInterrupt(ActionAbort{})
	ui.SetUndo(ActionUndo{})
	for _, binding := range globalControls() {
		ui.SetKey(binding.Key(), binding.Action())
	}
	for i := 0; i < 10; i++ {
		key := rune('0' + (i+1)%10)
		ui.SetKey(key, ActionSelectSuggestion{i})
	}
	var caser *cases.Caser
	if cfg.General.IgnoreCase {
		caser = new(cases.Caser)
		*caser = cases.Fold()
	}
	return SpellChecker{
		scr:          scr,
		ui:           ui,
		layout:       layout,
		speller:      speller,
		ignore:       make(map[string]bool),
		replacements: make(map[string]string),
		doBackup:     cfg.General.Backup || options.backup,
		caser:        caser,
	}
}

// transform applies case folding if enabled.
func (self *SpellChecker) transform(word string) string {
	if self.caser != nil {
		word = self.caser.String(word)
	}
	return word
}

// replaceAllInFile replaces all occurrences of a word in the current file.
// from should already be transformed.
func (self *SpellChecker) replaceAllInFile(file *FileContext, from string, to string, after tui.SliceIndex) {
	for _, word := range file.Source().Words() {
		if word.Index.IsAfter(after) && self.transform(word.Original) == from {
			file.Change(word.Index, to)
		}
	}
}

func (self *SpellChecker) setFile(id int) *FileContext {
	self.currentFile = id
	self.layout.SetSource(self.files[id].Source())
	return &self.files[id]
}

// AddFile adds a file to the checker
func (self *SpellChecker) AddFile(sf SourceFile) {
	self.files = append(self.files, NewFileContext(sf))
	self.currentFile = len(self.files) - 1
	file := &self.files[len(self.files)-1]
	for from, to := range self.replacements {
		self.replaceAllInFile(file, from, to, tui.NewSliceIndex(0, 0))
	}
}

func (self *SpellChecker) doUndo(event UndoEventBase) (int, int) {
	file := &self.files[self.currentFile]
	evFileId := event.fileId
	evWordId := event.wordId
	switch event := event.kind.(type) {
	case UndoReplacement:
		file.RemoveChange(event.slice, file.Word(evWordId).Original)

	case UndoIgnore:
		if event.all {
			delete(self.ignore, event.word)
		}

	case UndoSkip:

	case UndoReplaceAll:
		delete(self.replacements, event.from)
		for fileId := evFileId; fileId < len(self.files); fileId++ {
			start := tui.SliceIndex{}
			if fileId == evFileId {
				start = event.startIndex
			}
			file := self.files[fileId]
			for _, word := range file.Source().Words() {
				if word.Index.IsAfter(start) && self.transform(word.Original) == event.from {
					file.RemoveChange(word.Index, word.Original)
				}
			}
		}

	default:
		panic("not an undo event")
	}
	return evFileId, evWordId
}

// Run runs the checker until all current files are checked. Returns true if the
// program should quit.
func (self *SpellChecker) Run() bool {
	file := self.setFile(self.currentFile)
	fileEnd := len(self.files)
	wordEnd := len(file.Source().Words())
	fileId := self.currentFile
	wordId := 0
	addUndoEvent := func(kind any) {
		self.undoStack = append(
			self.undoStack,
			UndoEventBase{fileId, wordId, kind},
		)
	}
	for {
		// The blow code can freely change the word and files ids but should not
		// change the current file, we only handle all those changes here.
		if wordId >= wordEnd {
			wordId = 0
			fileId++
		}
		if fileId != self.currentFile {
			if fileId >= fileEnd {
				return false
			}
			file = self.setFile(fileId)
			wordEnd = len(file.Source().Words())
		}

		word := file.Word(wordId)
		if self.ignore[self.transform(word.Original)] || file.SliceIsChanged(word.Index) {
			wordId++
			continue
		}
		suggestions := self.speller.Suggest(word.Original)
		if len(suggestions) > 20 {
			suggestions = suggestions[:20]
		}
		self.layout.SetSuggestions(suggestions)
		self.layout.Show(word.Index)

	repeatKey:
		self.ui.Update(nil)
		switch action := self.ui.RunUntilAction().(type) {
		case ActionSelectSuggestion:
			// We always have all 10 keys bound so we need to ignore presses
			// if there aren't enough suggestions here.
			if action.index >= len(suggestions) {
				goto repeatKey
			}
			replacement := suggestions[action.index]
			file.Change(word.Index, replacement)
			self.speller.Replace(word.Original, replacement)
			addUndoEvent(UndoReplacement{word.Index})
			self.changed = true
			wordId++

		case ActionIgnore:
			var original string
			if action.all {
				original = self.transform(word.Original)
				self.ignore[original] = true
			}
			addUndoEvent(UndoIgnore{action.all, original})
			wordId++

		case ActionReplace:
			var caption string
			if action.all {
				caption = "Replace all"
			} else {
				caption = "Replace"
			}
			maybeText := tui.InputBox(
				self.scr,
				caption,
				"Enter replacement",
				self.speller.Suggest,
			)
			if maybeText.IsSome() && len(maybeText.Unwrap()) > 0 {
				text := maybeText.Unwrap()
				if action.all {
					original := self.transform(word.Original)
					self.replacements[original] = text
					for id := fileId; id < fileEnd; id++ {
						self.replaceAllInFile(&self.files[id], original, text, word.Index)
					}
					addUndoEvent(UndoReplaceAll{word.Index, original})
				} else {
					file.Change(word.Index, text)
					addUndoEvent(UndoReplacement{word.Index})
				}
				self.speller.Replace(word.Original, text)
				self.changed = true
				wordId++
			} else {
				goto repeatKey
			}

		case ActionUndo:
			if len(self.undoStack) == 0 {
				goto repeatKey
			}
			var event UndoEventBase
			event, self.undoStack = util.PopBack(self.undoStack)
			fileId, wordId = self.doUndo(event)

		case ActionSkip:
			addUndoEvent(UndoSkip{})
			fileId++

		case ActionAbort:
			if !self.changed ||
				tui.AskYesNo(self.scr, "Are you sure you want to abort?") {
				self.discardAll = true
				return true
			}
			goto repeatKey

		case ActionExit:
			return true
		}
	}
}

func (self *SpellChecker) Finish() {
	if self.discardAll || !self.changed {
		return
	}
	backup := Backup{}
	if self.doBackup {
		if err := backup.Create(); err != nil {
			Fatal("could not created backup: %s", err)
		}
	}
	for _, file := range self.files {
		if !file.IsChanged() {
			continue
		}
		if err := file.Write(); err != nil {
			log.Printf("%s: could not write %s: %s\n", InvocationName, file.sf.Name(), err)
		} else if self.doBackup {
			file.AddToBackup(&backup)
		}
	}
	if self.doBackup {
		backup.Write()
	}
}
