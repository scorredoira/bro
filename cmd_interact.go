package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

func cmdClick(ctx *cmdContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bro click [--css <sel>] [--id <id>] [text]")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	q := parseElementArgs(args)
	el, err := findElement(page, q)
	if err != nil {
		return err
	}

	// CSS/ID-found elements may be off-screen; use JS click to avoid
	// Rod's scroll-into-view which can hang on complex layouts.
	if q.css != "" || q.id != "" {
		_, err = el.Eval(`() => this.click()`)
	} else {
		err = el.Click(proto.InputMouseButtonLeft, 1)
	}
	if err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	fmt.Printf("clicked %s\n", describeQuery(q))
	return nil
}

func cmdDblClick(ctx *cmdContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bro dblclick [--css <sel>] [--id <id>] [text]")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	q := parseElementArgs(args)
	el, err := findElement(page, q)
	if err != nil {
		return err
	}

	if err := el.Click(proto.InputMouseButtonLeft, 2); err != nil {
		return fmt.Errorf("double-click failed: %w", err)
	}

	fmt.Printf("double-clicked %s\n", describeQuery(q))
	return nil
}

func cmdFill(ctx *cmdContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: bro fill <label> <value>")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	label := args[0]
	value := strings.Join(args[1:], " ")

	el, err := findInput(page, label)
	if err != nil {
		return err
	}

	// Clear existing value.
	_ = el.SelectAllText()
	page.Keyboard.Press(input.Backspace)

	if err := el.Input(value); err != nil {
		return fmt.Errorf("fill failed: %w", err)
	}

	fmt.Printf("filled %q = %q\n", label, value)
	return nil
}

func cmdSelect(ctx *cmdContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: bro select <label> <value>")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	label := args[0]
	value := strings.Join(args[1:], " ")

	// Try standard input roles first, then widget classes by CSS, then
	// any element with matching text as last resort.
	el, err := findInput(page, label)
	if err != nil {
		// Look for custom widget dropdowns (.Select, .Input) by label text.
		// This avoids findByText matching background elements behind modals.
		el, err = findByCSS(page, ".Select", label)
		if err != nil {
			el, err = findByText(page, label)
			if err != nil {
				return fmt.Errorf("input with label %q not found", label)
			}
		}
	}

	tag, _ := el.Eval(`() => this.tagName`)
	if tag != nil && strings.EqualFold(tag.Value.Str(), "select") {
		err = el.Select([]string{value}, true, rod.SelectorTypeText)
		if err != nil {
			return fmt.Errorf("select failed: %w", err)
		}
	} else {
		// For custom dropdowns: click to open, then click the option.
		// Use JS click to avoid Rod's ScrollIntoView which triggers
		// scroll events that close framework popups.
		if _, err := el.Eval(`() => this.click()`); err != nil {
			return fmt.Errorf("click to open failed: %w", err)
		}
		time.Sleep(300 * time.Millisecond)

		// Look for the option inside the visible popup first, to avoid
		// matching background elements behind modals/forms.
		option, err := findByCSS(page, ".Popup .row, .Popup .option, .Popup li", value)
		if err != nil {
			// Fall back to full AX tree search.
			option, err = findByText(page, value)
			if err != nil {
				return fmt.Errorf("option %q not found: %w", value, err)
			}
		}
		if _, err := option.Eval(`() => this.click()`); err != nil {
			return fmt.Errorf("click option failed: %w", err)
		}
	}

	fmt.Printf("selected %q = %q\n", label, value)
	return nil
}

func cmdType(ctx *cmdContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bro type <text>")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	text := strings.Join(args, " ")
	if err := page.InsertText(text); err != nil {
		return fmt.Errorf("type failed: %w", err)
	}

	fmt.Printf("typed %q\n", text)
	return nil
}

func cmdPress(ctx *cmdContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bro press <key>")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	key := resolveKey(args[0])
	if err := page.KeyActions().Press(key).Do(); err != nil {
		return fmt.Errorf("press failed: %w", err)
	}

	fmt.Printf("pressed %s\n", args[0])
	return nil
}

func cmdHover(ctx *cmdContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bro hover [--css <sel>] [--id <id>] [text]")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	q := parseElementArgs(args)
	el, err := findElement(page, q)
	if err != nil {
		return err
	}

	if err := el.Hover(); err != nil {
		return fmt.Errorf("hover failed: %w", err)
	}

	fmt.Printf("hovering %s\n", describeQuery(q))
	return nil
}

func cmdDrag(ctx *cmdContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: bro drag <from-text> <to-text>")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	from, err := findByText(page, args[0])
	if err != nil {
		return fmt.Errorf("from element: %w", err)
	}

	to, err := findByText(page, args[1])
	if err != nil {
		return fmt.Errorf("to element: %w", err)
	}

	fromBox, err := from.Shape()
	if err != nil {
		return fmt.Errorf("from shape: %w", err)
	}
	toBox, err := to.Shape()
	if err != nil {
		return fmt.Errorf("to shape: %w", err)
	}

	fromPt := fromBox.OnePointInside()
	toPt := toBox.OnePointInside()

	mouse := page.Mouse
	if err := mouse.MoveTo(proto.NewPoint(fromPt.X, fromPt.Y)); err != nil {
		return err
	}
	if err := mouse.Down(proto.InputMouseButtonLeft, 1); err != nil {
		return err
	}
	if err := mouse.MoveTo(proto.NewPoint(toPt.X, toPt.Y)); err != nil {
		return err
	}
	if err := mouse.Up(proto.InputMouseButtonLeft, 1); err != nil {
		return err
	}

	fmt.Printf("dragged %q -> %q\n", args[0], args[1])
	return nil
}

func cmdUpload(ctx *cmdContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: bro upload <selector> <filepath>")
	}

	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	el, err := page.Element(args[0])
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	if err := el.SetFiles([]string{args[1]}); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Printf("uploaded %s\n", args[1])
	return nil
}

func resolveKey(name string) input.Key {
	switch strings.ToLower(name) {
	case "enter":
		return input.Enter
	case "tab":
		return input.Tab
	case "escape", "esc":
		return input.Escape
	case "backspace":
		return input.Backspace
	case "delete":
		return input.Delete
	case "arrowup", "up":
		return input.ArrowUp
	case "arrowdown", "down":
		return input.ArrowDown
	case "arrowleft", "left":
		return input.ArrowLeft
	case "arrowright", "right":
		return input.ArrowRight
	case "space":
		return input.Space
	case "home":
		return input.Home
	case "end":
		return input.End
	case "pageup":
		return input.PageUp
	case "pagedown":
		return input.PageDown
	default:
		return input.Key(rune(name[0]))
	}
}
