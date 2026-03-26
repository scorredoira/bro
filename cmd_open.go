package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func cmdOpen(ctx *cmdContext, args []string, portSet bool) error {
	url := ""
	if len(args) > 0 {
		url = args[0]
	}

	// If port was explicitly set, use it. Otherwise find a free one.
	port := ctx.port
	if !portSet {
		port = findFreePort(ctx.port)
	}

	// If already running on this port, navigate if URL given.
	if isPortOpen(port) {
		fmt.Println(port)
		if url != "" {
			ctx.port = port
			return cmdNavigate(ctx, []string{url})
		}
		return nil
	}

	// Each port gets its own profile dir for full isolation.
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("bro-chrome-%d", port))

	// Disable password manager via Chrome preferences.
	defaultDir := filepath.Join(dataDir, "Default")
	os.MkdirAll(defaultDir, 0755)
	prefsPath := filepath.Join(defaultDir, "Preferences")
	prefs := `{"credentials_enable_service":false,"profile":{"password_manager_enabled":false},"translate":{"enabled":false},"translate_blocked_languages":["en","es","fr","de","pt","it","zh","ja","ko","ru","ar"]}`
	os.WriteFile(prefsPath, []byte(prefs), 0644)

	chromeArgs := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", dataDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-default-apps",
		"--disable-popup-blocking",
		"--disable-translate",
		"--disable-features=PasswordManagerOnboarding,PasswordManagerBubble,TranslateUI,Translate",
		"--disable-component-update",
		"--disable-extensions",
		"--window-size=1280,900",
	}

	if ctx.headless {
		chromeArgs = append(chromeArgs, "--headless=new")
	}

	chromePath, err := findChromePath()
	if err != nil {
		return err
	}

	if runtime.GOOS == "darwin" {
		chromeArgs = append(chromeArgs, "--password-store=basic", "--use-mock-keychain")
	}

	if url != "" {
		chromeArgs = append(chromeArgs, url)
	}

	chromeCmd := exec.Command(chromePath, chromeArgs...)
	if err := chromeCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Chrome: %w", err)
	}
	ctx.pid = chromeCmd.Process.Pid

	// Wait for Chrome to be ready.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if isPortOpen(port) {
			// Start background monitor to kill Chrome when all windows are closed.
			if !ctx.headless {
				startMonitor(port, chromeCmd.Process.Pid)
			}

			// If a URL was given, wait for the page to finish loading.
			if url != "" {
				ctx.port = port
				_, page, err := connect(ctx)
				if err == nil {
					page.Timeout(10 * time.Second).WaitLoad()
				}
			}

			fmt.Println(port)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("Chrome started but port %d not ready after 10s", port)
}

func cmdClose(ctx *cmdContext) error {
	if !isPortOpen(ctx.port) {
		return fmt.Errorf("no Chrome running on port %d", ctx.port)
	}

	browser, err := connectBrowser(ctx)
	if err != nil {
		return err
	}

	if err := browser.Close(); err != nil {
		return fmt.Errorf("failed to close Chrome: %w", err)
	}

	fmt.Printf("Chrome on port %d closed\n", ctx.port)
	return nil
}

func findChromePath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", nil
	case "linux":
		path := findInPath("google-chrome", "google-chrome-stable", "chromium", "chromium-browser")
		if path == "" {
			return "", fmt.Errorf("Chrome not found. Install Google Chrome or Chromium")
		}
		return path, nil
	case "windows":
		path := findInPath(
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		)
		if path == "" {
			path = findInPath("chrome")
		}
		if path == "" {
			return "", fmt.Errorf("Chrome not found. Install Google Chrome")
		}
		return path, nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// findFreePort finds the first available port starting from base.
func findFreePort(base int) int {
	for port := base; port < base+100; port++ {
		if !isPortOpen(port) {
			return port
		}
	}
	return base + 100
}

func isPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// startMonitor spawns a background process that watches Chrome
// and kills it when all windows are closed.
func startMonitor(port, pid int) {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	m := exec.Command(exePath, "_monitor", strconv.Itoa(port), strconv.Itoa(pid))
	m.Stdout = nil
	m.Stderr = nil
	m.Start()
}

// cmdMonitor runs in the background, polling Chrome's targets.
// When no pages remain (user closed all windows), it kills Chrome.
func cmdMonitor(args []string) {
	if len(args) < 2 {
		return
	}
	port, _ := strconv.Atoi(args[0])
	pid, _ := strconv.Atoi(args[1])

	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	// Give Chrome time to start and load the first page.
	time.Sleep(5 * time.Second)

	for {
		time.Sleep(2 * time.Second)

		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/json/list", port))
		if err != nil {
			// Chrome is gone.
			return
		}

		var targets []chromeTarget
		json.NewDecoder(resp.Body).Decode(&targets)
		resp.Body.Close()

		hasPages := false
		for _, t := range targets {
			if t.Type == "page" && !strings.HasPrefix(t.URL, "chrome://") {
				hasPages = true
				break
			}
		}

		if !hasPages {
			proc.Kill()
			return
		}
	}
}

func findInPath(names ...string) string {
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err == nil {
			return path
		}
	}
	return ""
}
