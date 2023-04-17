package main

import (
	"log"

	. "github.com/JaMo42/spellcheck_comments/common"
	. "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/gdamore/tcell/v2"
	"github.com/trustmaster/go-aspell"
)

type ActionIgnore struct{ all bool }
type ActionReplace struct{ all bool }
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
	scr        tcell.Screen
	ui         tui.Tui
	layout     Layout
	speller    aspell.Speller
	ignore     map[string]bool
	changed    bool
	discardAll bool
	doBackup   bool
	files      []FileContext
}

func NewSpellChecker(
	scr tcell.Screen, speller aspell.Speller, cfg *Config, options *Options,
) SpellChecker {
	var layout Layout
	switch cfg.General.Layout {
	case "aspell":
		panic("unimplemented")
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
	return SpellChecker{
		scr:      scr,
		ui:       ui,
		layout:   layout,
		speller:  speller,
		ignore:   make(map[string]bool),
		doBackup: cfg.General.Backup || options.backup,
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
	for maybeWord := sf.NextWord(); maybeWord.IsSome(); maybeWord = sf.NextWord() {
		word := maybeWord.Get()
		if self.ignore[word.Original] {
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
			file.Change(word.Index, suggestions[action.index])
			self.changed = true

		case ActionIgnore:
			if action.all {
				self.ignore[word.Original] = true
			}

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
