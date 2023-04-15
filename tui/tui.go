package tui

import (
	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/gdamore/tcell/v2"
)

type ArrowReceiver interface {
	Up()
	Down()
	Left()
	Right()
	GetSelected() any
}

type MouseReceiver interface {
	Motion(x, y int) bool
	Click(button tcell.ButtonMask) Optional[any]
}

type Layout interface {
	Create()
	Layout(width, height int)
	Update(scr tcell.Screen)
}

type KeyAction interface {
	Key() rune
	Label() string
	Action() any
}

type Tui struct {
	scr       tcell.Screen
	layout    Layout
	arrow     ArrowReceiver
	mouse     []MouseReceiver
	keys      map[rune]any
	interrupt any
}

func NewTui(scr tcell.Screen, layout Layout) Tui {
	layout.Create()
	layout.Layout(scr.Size())
	return Tui{
		scr:    scr,
		layout: layout,
		keys:   make(map[rune]any),
	}
}

// Layout calls the layouts Layout method with the current screen size.
func (self *Tui) Layout() {
	self.layout.Layout(self.scr.Size())
}

// Update calls the layouts Update method and shows the screen.
func (self *Tui) Update() {
	self.layout.Update(self.scr)
	self.scr.Show()
}

func (self *Tui) SetKey(key rune, action any) {
	self.keys[key] = action
}

func (self *Tui) SetKeys(keys []rune, action any) {
	for _, k := range keys {
		self.keys[k] = action
	}
}

func (self *Tui) RemoveKey(key rune) {
	delete(self.keys, key)
}

func (self *Tui) SetInterrupt(action any) {
	self.interrupt = action
}

func (self *Tui) SetArrowReceiver(receiver ArrowReceiver) {
	self.arrow = receiver
}

func (self *Tui) SetMouseReceivers(receivers []MouseReceiver) {
	self.mouse = receivers
}

func (self *Tui) RunUntilAction() any {
	var action Optional[any]
	for {
		ev := self.scr.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			action = self.keyEvent(ev)
		case *tcell.EventMouse:
			action = self.mouseEvent(ev)
		case *tcell.EventResize:
			self.Layout()
			self.Update()
		}
		if action.IsSome() {
			return action.Unwrap()
		}
	}
}

func (self *Tui) keyEvent(ev *tcell.EventKey) Optional[any] {
	k, r := TranslateControls(ev)
	switch k {
	case tcell.KeyUp:
		self.arrow.Up()
	case tcell.KeyDown:
		self.arrow.Down()
	case tcell.KeyLeft:
		self.arrow.Left()
	case tcell.KeyRight:
		self.arrow.Right()
	case tcell.KeyEnter:
		return Some(self.arrow.GetSelected())
	case tcell.KeyRune:
		action, found := self.keys[r]
		if found {
			return Some(action)
		}
	case tcell.KeyCtrlC:
		return Some(self.interrupt)
	}
	return None[any]()
}

func (self *Tui) mouseEvent(ev *tcell.EventMouse) Optional[any] {
	if ev.Buttons() == 0 {
		update := false
		for _, receiver := range self.mouse {
			if receiver.Motion(ev.Position()) {
				update = true
			}
		}
		if update {
			self.Update()
		}
	} else {
		for _, receiver := range self.mouse {
			if maybeAction := receiver.Click(ev.Buttons()); maybeAction.IsSome() {
				return Some(maybeAction.Unwrap())
			}
		}
	}
	return None[any]()
}
