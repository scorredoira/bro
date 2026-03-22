package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

const retryTimeout = 3 * time.Second

// Interactive roles, in priority order.
var interactiveRoles = map[string]int{
	"button":           1,
	"link":             2,
	"menuitem":         3,
	"tab":              4,
	"checkbox":         5,
	"radio":            6,
	"switch":           7,
	"combobox":         8,
	"textbox":          9,
	"searchbox":        10,
	"option":           11,
	"menuitemcheckbox": 12,
	"menuitemradio":    13,
}

// findByText finds an element by its name in the accessibility tree.
// Retries for up to 3 seconds if not found immediately.
func findByText(page *rod.Page, text string) (*rod.Element, error) {
	deadline := time.Now().Add(retryTimeout)

	for {
		tree, err := proto.AccessibilityGetFullAXTree{}.Call(page)
		if err != nil {
			return nil, fmt.Errorf("accessibility tree failed: %w", err)
		}

		node := findAXNode(tree.Nodes, text, true)
		if node == nil {
			node = findAXNode(tree.Nodes, text, false)
		}
		if node != nil {
			return resolveAXNode(page, node)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("element with text %q not found", text)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// findInput finds an input element by its label in the accessibility tree.
// Retries for up to 3 seconds if not found immediately.
func findInput(page *rod.Page, label string) (*rod.Element, error) {
	lower := strings.ToLower(label)
	deadline := time.Now().Add(retryTimeout)

	for {
		tree, err := proto.AccessibilityGetFullAXTree{}.Call(page)
		if err != nil {
			return nil, fmt.Errorf("accessibility tree failed: %w", err)
		}

		for _, node := range tree.Nodes {
			if node.Ignored {
				continue
			}
			role := axValueStr(node.Role)
			name := axValueStr(node.Name)

			switch role {
			case "textbox", "searchbox", "combobox", "spinbutton", "slider", "textarea":
				if matchesLabel(name, lower) {
					return resolveAXNode(page, node)
				}
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("input with label %q not found", label)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// findAXNode finds a node in the accessibility tree by name.
// Prefers interactive elements over static text.
func findAXNode(nodes []*proto.AccessibilityAXNode, text string, exact bool) *proto.AccessibilityAXNode {
	lower := strings.ToLower(text)

	var best *proto.AccessibilityAXNode
	bestPriority := 999

	for _, node := range nodes {
		if node.Ignored || node.BackendDOMNodeID == 0 {
			continue
		}
		name := stripZeroWidth(axValueStr(node.Name))
		if name == "" {
			continue
		}

		matched := false
		if exact {
			matched = strings.EqualFold(name, text)
		} else {
			matched = strings.Contains(strings.ToLower(name), lower)
		}
		if !matched {
			continue
		}

		role := axValueStr(node.Role)
		priority, isInteractive := interactiveRoles[role]
		if isInteractive {
			if priority < bestPriority {
				best = node
				bestPriority = priority
			}
		} else if best == nil {
			best = node
		}
	}

	return best
}

// resolveAXNode converts an accessibility tree node to a Rod element.
// If the node resolves to a Text node (e.g. StaticText / InlineTextBox),
// it walks up to the nearest parent Element so callers can safely click,
// hover, or call getComputedStyle on it.
func resolveAXNode(page *rod.Page, node *proto.AccessibilityAXNode) (*rod.Element, error) {
	if node.BackendDOMNodeID == 0 {
		return nil, fmt.Errorf("node %q has no DOM backing", axValueStr(node.Name))
	}

	result, err := proto.DOMResolveNode{
		BackendNodeID: node.BackendDOMNodeID,
	}.Call(page)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DOM node: %w", err)
	}

	el, err := page.ElementFromObject(result.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to create element: %w", err)
	}

	// Text nodes (nodeType 3) cannot be clicked or styled.
	// Walk up to the parent element.
	nodeType, _ := el.Eval(`() => this.nodeType`)
	if nodeType != nil && nodeType.Value.Int() == 3 {
		parent, err := el.Parent()
		if err != nil {
			return nil, fmt.Errorf("failed to get parent of text node: %w", err)
		}
		return parent, nil
	}

	return el, nil
}

func matchesLabel(name, lowerLabel string) bool {
	if name == "" {
		return false
	}
	nameLower := strings.ToLower(stripZeroWidth(name))
	nameLower = strings.TrimRight(nameLower, " *")
	return nameLower == lowerLabel || strings.Contains(nameLower, lowerLabel)
}

// stripZeroWidth removes zero-width characters that some frameworks
// insert between letters in placeholder text (e.g. "N\u200bo\u200bm\u200bb\u200br\u200be").
func stripZeroWidth(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\ufeff':
			return -1
		}
		return r
	}, s)
}

// findByCSS finds an element by CSS selector, optionally filtered by text content.
// Uses RuntimeEvaluate for fast single-call DOM queries.
// Retries for up to 3 seconds if not found immediately.
func findByCSS(page *rod.Page, selector, textFilter string) (*rod.Element, error) {
	deadline := time.Now().Add(retryTimeout)

	var jsCode string
	if textFilter == "" {
		jsCode = fmt.Sprintf(`document.querySelector(%q)`, selector)
	} else {
		jsCode = fmt.Sprintf(`(function(){
			var lower = %q.toLowerCase();
			var els = document.querySelectorAll(%q);
			for (var i = 0; i < els.length; i++) {
				var t = els[i].textContent.trim();
				if (t.toLowerCase() === lower || t.toLowerCase().indexOf(lower) !== -1) {
					return els[i];
				}
			}
			return null;
		})()`, textFilter, selector)
	}

	for {
		res, err := proto.RuntimeEvaluate{Expression: jsCode}.Call(page)
		if err == nil && res.Result != nil && res.Result.ObjectID != "" {
			el, err := page.ElementFromObject(res.Result)
			if err == nil {
				return el, nil
			}
		}

		if time.Now().After(deadline) {
			if textFilter != "" {
				return nil, fmt.Errorf("no element matching %q with text %q", selector, textFilter)
			}
			return nil, fmt.Errorf("no element matching %q", selector)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// findByID finds an element by its DOM id attribute.
// Retries for up to 3 seconds if not found immediately.
func findByID(page *rod.Page, id string) (*rod.Element, error) {
	return findByCSS(page, "#"+id, "")
}

// elementQuery holds parsed flags for element finding.
type elementQuery struct {
	css        string // --css selector
	id         string // --id selector
	textFilter string // remaining positional args (text filter)
}

// parseElementArgs extracts --css and --id flags from args.
// Remaining args are joined as the text filter.
func parseElementArgs(args []string) elementQuery {
	var q elementQuery
	var rest []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--css":
			if i+1 < len(args) {
				q.css = args[i+1]
				i++
			}
		case "--id":
			if i+1 < len(args) {
				q.id = args[i+1]
				i++
			}
		default:
			rest = append(rest, args[i])
		}
	}

	q.textFilter = strings.Join(rest, " ")
	return q
}

// findElement finds an element using the parsed query.
// Priority: --css > --id > text (AX tree).
func findElement(page *rod.Page, q elementQuery) (*rod.Element, error) {
	if q.css != "" {
		return findByCSS(page, q.css, q.textFilter)
	}
	if q.id != "" {
		return findByID(page, q.id)
	}
	if q.textFilter == "" {
		return nil, fmt.Errorf("no selector or text provided")
	}
	return findByText(page, q.textFilter)
}

// describeQuery returns a human-readable description of how the element was found.
func describeQuery(q elementQuery) string {
	if q.css != "" && q.textFilter != "" {
		return fmt.Sprintf("css %q text %q", q.css, q.textFilter)
	}
	if q.css != "" {
		return fmt.Sprintf("css %q", q.css)
	}
	if q.id != "" {
		return fmt.Sprintf("id %q", q.id)
	}
	return fmt.Sprintf("%q", q.textFilter)
}
