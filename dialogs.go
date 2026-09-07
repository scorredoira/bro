package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// dialogPolicy is how this run answers the dialogs the page opens, and which pages already
// have somebody listening.
//
// The zero value is the default — see answer(). `bro dialog accept` flips it, and the choice
// holds for the rest of the process, which is what a .bro test file is.
//
// The lock is not decoration: the listener is a goroutine that outlives the command that
// started it, so it reads this while a later command is writing it.
type dialogPolicy struct {
	mu       sync.Mutex
	explicit bool
	accept   bool
	text     string
	armed    map[proto.TargetTargetID]bool
}

func (p *dialogPolicy) set(accept bool, text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.explicit = true
	p.accept = accept
	p.text = text
}

// answer says how a dialog of this kind gets closed.
//
// A `beforeunload` is not a question about what the app should do: it is the page asking
// whether to let go of edits this run itself typed, and the command it interrupted is the one
// that asked to leave. Answering anything but yes cancels a navigation that was ordered.
//
// An `alert` has one button. Accept and dismiss both close it; accept is what a user does.
//
// A `confirm` or a `prompt` is a real question, and the default answer is NO. Pressing OK on
// "Delete everything?" while nobody is looking is not something a tool should do on its own —
// `bro dialog accept` is how you say otherwise.
func (p *dialogPolicy) answer(kind proto.PageDialogType) (bool, string) {
	if kind == proto.PageDialogTypeBeforeunload || kind == proto.PageDialogTypeAlert {
		return true, ""
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.explicit {
		return false, ""
	}
	return p.accept, p.text
}

// arm claims a page for the listener, and says whether it was already claimed.
func (p *dialogPolicy) arm(target proto.TargetTargetID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.armed[target] {
		return false
	}
	if p.armed == nil {
		p.armed = map[proto.TargetTargetID]bool{}
	}
	p.armed[target] = true
	return true
}

// armDialogs answers the page's dialogs, because there is nobody at the keyboard to do it.
//
// A native dialog — an alert, a confirm, or the "Leave site? Changes you made may not be
// saved" that an unsaved-changes guard raises on navigate — blocks the page until somebody
// presses a button. bro is a CLI: every command is a process that connects, acts and exits, so
// a dialog raised by a click is still standing when that process is gone. The command that
// opened it hangs, every command after it times out against a page that cannot answer, and the
// only way out is a human walking to the machine and clicking. That is what this stops.
//
// Armed in connect(), so it covers the command that opens the dialog rather than a later one:
// by the time a second command runs, the first has already hung.
//
// It does NOT press OK on everything: what each kind gets is in answer(), and a confirm is
// still answered no unless you said otherwise.
//
// Once per page, not once per command: in a .bro test file every command connects again over
// the same tab, and a listener per connection means every listener but the first finds
// the dialog already answered and reports a failure that is only its own duplication.
//
// What it answered is printed, never swallowed. A confirm answered behind your back changes
// what the app did, and a test that passes without saying so is worse than one that hangs.
func armDialogs(page *rod.Page, policy *dialogPolicy) {
	if !policy.arm(page.TargetID) {
		return
	}

	wait := page.EachEvent(func(e *proto.PageJavascriptDialogOpening) bool {
		accept, text := policy.answer(e.Type)
		err := proto.PageHandleJavaScriptDialog{
			Accept:     accept,
			PromptText: text,
		}.Call(page)

		answer := "accepted"
		if !accept {
			answer = "dismissed"
		}
		if err != nil {
			answer = fmt.Sprintf("could not be answered: %v", err)
		}
		fmt.Fprintf(os.Stderr, "dialog: %s %q → %s\n", e.Type, e.Message, answer)

		// Keep listening: a page that asks once usually asks again.
		return false
	})

	go wait()
}

// dialogPolicy returns this run's policy, creating it if no command has connected yet — a
// `dialog` line can be the first line of a test file.
func (ctx *cmdContext) dialogPolicy() *dialogPolicy {
	if ctx.dialogs == nil {
		ctx.dialogs = &dialogPolicy{}
	}
	return ctx.dialogs
}
