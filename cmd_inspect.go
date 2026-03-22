package main

import (
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func cmdSnapshot(ctx *cmdContext, args []string) error {
	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	verbose := len(args) > 0 && args[0] == "--verbose"

	tree, err := getAccessibilityTree(page, verbose)
	if err != nil {
		return err
	}

	fmt.Print(tree)
	return nil
}

func getAccessibilityTree(page *rod.Page, verbose bool) (string, error) {
	result, err := proto.AccessibilityGetFullAXTree{}.Call(page)
	if err != nil {
		return "", fmt.Errorf("accessibility tree failed: %w", err)
	}

	var sb strings.Builder
	for _, node := range result.Nodes {
		role := axValueStr(node.Role)
		name := axValueStr(node.Name)
		value := axValueStr(node.Value)

		if node.Ignored {
			continue
		}

		if !verbose {
			if !isUsefulRole(role) {
				continue
			}
		}

		line := fmt.Sprintf("[%s] %s", node.NodeID, role)
		if name != "" {
			line += fmt.Sprintf(" %q", name)
		}
		if value != "" {
			line += fmt.Sprintf(" value=%q", value)
		}

		if verbose && node.Properties != nil {
			for _, prop := range node.Properties {
				if prop.Value != nil {
					line += fmt.Sprintf(" %s=%s", prop.Name, prop.Value.Value.Str())
				}
			}
		}

		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String(), nil
}

func axValueStr(v *proto.AccessibilityAXValue) string {
	if v == nil {
		return ""
	}
	return v.Value.Str()
}

var usefulRoles = map[string]bool{
	"button":       true,
	"link":         true,
	"textbox":      true,
	"searchbox":    true,
	"combobox":     true,
	"checkbox":     true,
	"radio":        true,
	"switch":       true,
	"tab":          true,
	"menuitem":     true,
	"option":       true,
	"slider":       true,
	"spinbutton":   true,
	"heading":      true,
	"RootWebArea":  true,
}

func isUsefulRole(role string) bool {
	return usefulRoles[role]
}

func cmdScreenshot(ctx *cmdContext, args []string) error {
	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	path := "/tmp/bro.png"
	fullPage := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--full":
			fullPage = true
		default:
			path = args[i]
		}
	}

	var data []byte
	if fullPage {
		data, err = page.Screenshot(true, nil)
	} else {
		data, err = page.Screenshot(false, nil)
	}
	if err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}

	if err := writeFile(path, data); err != nil {
		return err
	}

	fmt.Println(path)
	return nil
}

func cmdURL(ctx *cmdContext) error {
	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("failed to get page info: %w", err)
	}

	fmt.Println(info.URL)
	return nil
}

func cmdHTML(ctx *cmdContext) error {
	_, page, err := connect(ctx)
	if err != nil {
		return err
	}

	html, err := page.HTML()
	if err != nil {
		return fmt.Errorf("failed to get HTML: %w", err)
	}

	fmt.Println(html)
	return nil
}
