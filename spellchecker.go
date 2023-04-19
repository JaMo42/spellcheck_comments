package main

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/trustmaster/go-aspell"
	"golang.org/x/text/cases"

	. "github.com/JaMo42/spellcheck_comments/common"
	. "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
)

type ActionIgnore struct{ all bool }
type ActionReplace struct{ all bool }
type ActionSkip struct{}
type ActionExit struct{}
type ActionAbort struct{}
type ActionSelectSuggestion struct{ index int }

type Layout interface {
	SetSource(*SourceFile)
	Show(tui.SliceIndex)
	SetSuggestions([]string)
	ArrowReceiver() tui.ArrowReceiver
	MouseReceivers() []tui.MouseReceiver
}

type SpellChecker struct {
	scr     tcell.Screen
	ui      tui.Tui
	layout  Layout
	speller aspell.Speller
	ignore  map[string]bool
	// replacements holds the words changed with replaceAll
	replacements map[string]string
	// replaced holds the words changed by replaceAll in the current file
	replaced   map[tui.SliceIndex]bool
	changed    bool
	discardAll bool
	doBackup   bool
	files      []FileContext
	caser      *cases.Caser
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
		replaced:     make(map[tui.SliceIndex]bool),
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
func (self *SpellChecker) replaceAllInFile(file *FileContext, from string, to string, after tui.SliceIndex) {
	from = self.transform(from)
	for _, word := range file.Source().Words() {
		if word.Index.IsAfter(after) && self.transform(word.Original) == from {
			self.replaced[word.Index] = true
			file.Change(word.Index, to)
		}
	}
}

func (self *SpellChecker) CheckFile(sf SourceFile) bool {
	self.layout.SetSource(&sf)
	file := NewFileContext(sf)
	defer func() {
		if file.IsChanged() {
			self.files = append(self.files, file)
		}
	}()
	self.replaced = make(map[tui.SliceIndex]bool)
	for from, to := range self.replacements {
		self.replaceAllInFile(&file, from, to, tui.NewSliceIndex(0, 0))
	}
	for maybeWord := sf.NextWord(); maybeWord.IsSome(); maybeWord = sf.NextWord() {
		word := maybeWord.Get()
		if self.ignore[self.transform(word.Original)] || self.replaced[word.Index] {
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
			self.changed = true

		case ActionIgnore:
			if action.all {
				self.ignore[self.transform(word.Original)] = true
			}

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
					self.replacements[word.Original] = text
					self.replaceAllInFile(&file, word.Original, text, word.Index)
				} else {
					file.Change(word.Index, text)
				}
				self.speller.Replace(word.Original, text)
				self.changed = true
			} else {
				goto repeatKey
			}

		case ActionSkip:
			return false

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
	return false
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
