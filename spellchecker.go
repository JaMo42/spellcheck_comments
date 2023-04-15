package main

import (
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
	scr         tcell.Screen
	ui          tui.Tui
	layout      Layout
	speller     aspell.Speller
	ignore      map[string]bool
	currentWord Word
}

func NewSpellChecker(scr tcell.Screen, speller aspell.Speller, cfg *Config) SpellChecker {
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
	for _, binding := range GlobalControls() {
		ui.SetKey(binding.Key(), binding.Action())
	}
	return SpellChecker{
		scr:     scr,
		ui:      ui,
		layout:  layout,
		speller: speller,
	}
}

func (self *SpellChecker) Ignore(word string) bool {
	return self.ignore[word]
}

func (self *SpellChecker) AddIgnored(word string) {
	self.ignore[word] = true
}

func (self *SpellChecker) CheckFile(sf *SourceFile) {
	self.layout.SetSource(sf)
	for maybeWord := sf.NextWord(); maybeWord.IsSome(); maybeWord = sf.NextWord() {
		word := maybeWord.Get()
		self.layout.Show(word.Index)
		suggestions := self.speller.Suggest(word.Original)
		if len(suggestions) > 20 {
			suggestions = suggestions[:20]
		}
		self.layout.SetSuggestions(suggestions)
		self.ui.Layout()
	repeatKey:
		self.scr.Clear()
		self.ui.Update()
		switch action := self.ui.RunUntilAction().(type) {
		case ActionAbort:
			choice := tui.MessageBox(self.scr, "Are you sure you want to abort?", tui.MbYesNo)
			if choice == tui.MbYes {
				return
			}
			goto repeatKey
		case ActionIgnore:
			if action.all {
				self.AddIgnored(word.Original)
			}
		}
	}
}
