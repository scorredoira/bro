# bro — browser remote operator

CLI tool to control Chrome via DevTools Protocol. No MCP, no Puppeteer, no intermediaries. Talks directly to Chrome.

## Install

```bash
go install github.com/scorredoira/bro@latest
```

## Quick Start

Single session (most common):

```bash
bro open http://localhost:9092/admin/login
bro fill Email admin@demo.com
bro fill Password 123
bro click Login
bro close
```

Multiple parallel sessions (each gets its own isolated Chrome):

```bash
PORT=$(bro open http://localhost:9092/admin/login)
bro --port $PORT fill Email admin@demo.com
bro --port $PORT click Login
bro --port $PORT close
```

## Usage

```
bro [--port PORT] [--headless] [-w N] <command> [args...]
```

Default port: `9222`. Use `--headless` for headless mode (no browser window). When running multiple sessions, `bro open` auto-picks the next free port and prints it.

---

## Commands

### Chrome

#### open

Launch a new Chrome instance and optionally navigate to a URL. Prints the port number to stdout. Each instance is fully isolated (separate profile, cookies, storage).

```bash
bro open                                     # launch Chrome, print port
bro open http://localhost:9092/admin/login   # launch and navigate
bro --headless open http://example.com       # headless mode (no window)
```

If a Chrome is already running on the default port (or the one specified with `--port`), prints that port.

Works on macOS, Linux, and Windows. Auto-detects Chrome location.

#### close

Kill the Chrome instance on the given port.

```bash
bro close                    # close Chrome on default port 9222
bro --port 9223 close        # close Chrome on port 9223
```

---

### Navigation

#### navigate (alias: nav)

Go to a URL. Waits for the page to load.

```bash
bro --port $PORT navigate http://localhost:9092/admin/login
bro --port $PORT nav http://example.com
```

#### reload

Reload the current page.

```bash
bro --port $PORT reload
```

#### back

Go back in browser history.

```bash
bro --port $PORT back
```

#### forward

Go forward in browser history.

```bash
bro --port $PORT forward
```

#### resize

Resize the browser window.

```bash
bro --port $PORT resize 1280 720
bro --port $PORT resize 375 812    # iPhone X
bro --port $PORT resize 1920 1080  # Full HD
```

---

### Inspection

#### snapshot (alias: snap)

Print the accessibility tree. This is how you find elements to interact with.

```bash
bro --port $PORT snapshot
bro --port $PORT snap --verbose
```

Output:

```
[4] RootWebArea "Login"
[12] textbox "Email" value=""
[15] textbox "Password" value=""
[18] button "Login"
```

Use the text shown in quotes to target elements with `click`, `fill`, etc.

#### screenshot (alias: ss)

Take a screenshot. Defaults to `/tmp/bro.png`.

```bash
bro --port $PORT screenshot
bro --port $PORT ss /tmp/login-page.png
bro --port $PORT screenshot --full /tmp/full-page.png
```

Options:
- `--full` — capture the entire page (scroll included)

#### url

Print the current page URL.

```bash
bro --port $PORT url
```

#### html

Print the page HTML source.

```bash
bro --port $PORT html
bro --port $PORT html > /tmp/page.html
```

---

### Interaction

#### click

Click an element by its visible text. Also supports `--css` and `--id` flags for DOM-based lookup.

```bash
bro --port $PORT click Login
bro --port $PORT click "Save changes"

# Click by CSS selector (useful for elements not in the accessibility tree)
bro --port $PORT click --css ".grid-cell" "08:36"
bro --port $PORT click --css ".btn-primary"

# Click by DOM id
bro --port $PORT click --id submitBtn
```

When using `--css` or `--id`, an optional text argument filters by text content (case-insensitive substring match).

#### dblclick

Double-click an element. Supports `--css` and `--id` flags.

```bash
bro --port $PORT dblclick "row content"
bro --port $PORT dblclick --css ".editable-cell" "Total"
```

#### fill

Fill an input field. First argument is the label text, rest is the value. Matches by accessible name: `<label>`, `aria-label`, or **placeholder text**.

```bash
bro --port $PORT fill Email admin@demo.com
bro --port $PORT fill Password 123
bro --port $PORT fill "First name" Santiago
```

Zero-width characters in placeholder text are stripped automatically for matching.

#### select

Select a dropdown option. Works with native `<select>` and **custom widget dropdowns**. For custom widgets, it clicks to open the dropdown, waits for options to render, then clicks the matching option.

```bash
bro --port $PORT select Country Spain
bro --port $PORT select "Leave type" Vacation

# Works with custom widget dropdowns (e.g. React Select, S.Select)
bro --port $PORT select "Booking type" "Green Fee 18"
```

If the dropdown trigger doesn't have a standard input role (textbox, combobox), `select` falls back to finding it by visible text.

#### type

Type raw text into the currently focused element.

```bash
bro --port $PORT type "hello world"
```

#### press

Press a keyboard key.

```bash
bro --port $PORT press Enter
bro --port $PORT press Tab
bro --port $PORT press Escape
```

Supported keys: `Enter`, `Tab`, `Escape`/`Esc`, `Backspace`, `Delete`, `ArrowUp`/`Up`, `ArrowDown`/`Down`, `ArrowLeft`/`Left`, `ArrowRight`/`Right`, `Space`, `Home`, `End`, `PageUp`, `PageDown`.

#### hover

Hover over an element. Supports `--css` and `--id` flags.

```bash
bro --port $PORT hover "Settings"
bro --port $PORT hover --css ".menu-item" "Reports"
```

#### drag

Drag one element to another.

```bash
bro --port $PORT drag "Item 1" "Drop zone"
```

#### upload

Upload a file to a file input. Uses a CSS selector (not text).

```bash
bro --port $PORT upload "input[type=file]" /path/to/document.pdf
```

---

### Waits

#### wait

Wait for text to appear on the page (default timeout: 10s).

```bash
bro --port $PORT wait Dashboard
bro --port $PORT wait "Record saved"
bro --port $PORT wait --timeout 30s "Processing complete"
```

#### wait --gone

Wait for text to disappear.

```bash
bro --port $PORT wait --gone "Loading..."
```

#### wait --url

Wait for the URL to contain a pattern.

```bash
bro --port $PORT wait --url /admin/dashboard
```

---

### Tabs

#### pages

List all open tabs.

```bash
bro --port $PORT pages
```

#### page

Switch to a tab by its index.

```bash
bro --port $PORT page 0
```

#### newpage

Open a new tab.

```bash
bro --port $PORT newpage http://localhost:9092/admin
```

#### closepage

Close the current tab.

```bash
bro --port $PORT closepage
```

---

### JavaScript

#### js

Evaluate arbitrary JavaScript. Promises are automatically awaited.

```bash
bro --port $PORT js "document.title"
bro --port $PORT js "document.querySelectorAll('button').length"

# Promises are awaited automatically
bro --port $PORT js "fetch('/api/status').then(r => r.json())"
```

---

### Debug

#### console

Show captured console messages. On first call, installs a capture hook.

```bash
bro --port $PORT console    # installs capture
# ... do something ...
bro --port $PORT console    # shows messages
```

#### network (alias: net)

Show recent network requests.

```bash
bro --port $PORT network
```

---

### Dialogs

#### dialog accept

Accept the next JavaScript dialog.

```bash
bro --port $PORT dialog accept
bro --port $PORT dialog accept "prompt text"
```

#### dialog dismiss

Dismiss the next dialog.

```bash
bro --port $PORT dialog dismiss
```

**Note:** Call `dialog` *before* triggering the action that opens the dialog.

---

### Testing

#### test

Run `.bro` test files. Each test launches its own isolated Chrome, executes commands in order, and reports pass/fail.

```bash
bro test tests/                          # run all .bro files in directory (recursive)
bro test tests/login.bro                 # run a single test
bro --headless test tests/               # headless mode (no browser window)
bro -w 4 --headless test tests/          # 4 tests in parallel
```

---

## Test Files

Test files use the `.bro` extension. Each file is a sequence of bro commands, one per line:

```
# Login with valid credentials

open http://localhost:9092/admin/login
fill Email admin@demo.com
fill Password 123
click Login

assert url /admin/dashboard
assert text Dashboard
assert gone "Invalid credentials"
```

Rules:
- Lines starting with `#` are comments — the first one is the test name
- Blank lines are ignored
- Each line is a bro command (same syntax as the CLI)
- `assert` commands verify conditions with automatic retry (default timeout: 10s)

### Assert Commands

| Command | What it checks |
|---------|---------------|
| `assert url <pattern>` | URL contains pattern |
| `assert text <text>` | Text is visible on the page |
| `assert gone <text>` | Text is NOT on the page |
| `assert title <text>` | Page title contains text |
| `assert js <expression>` | JavaScript expression returns truthy |

Assertions retry automatically until they pass or timeout. No need for explicit waits or sleeps between actions and assertions.

Override the default 10s timeout:

```
click "Generate report"
assert --timeout 30s text "Report ready"
```

### Background Servers

`start` ensures a server is running before the test continues. If the port is already open, it's a no-op. Otherwise it launches the command and waits up to 30s for the port to accept HTTP connections.

```
start :3000 node server.js
start :9092 go run ./cmd/server
open http://localhost:3000
```

### Shell Commands

`exec` runs a shell command inside a test. Stdout is captured in `${result}` (trimmed).

```
# Read a token from the database and use it
exec mysql -N -s -e "SELECT token FROM s_main.accounts WHERE email='user@test.com'"
navigate http://localhost:9092/reset-password?token=${result}
```

Rules:
- Fails the test if exit code ≠ 0
- Stdout is always captured in `${result}` (overwrites previous value)
- `${result}` is expanded in all subsequent lines
- Use `exec --as VARNAME` for named variables: `exec --as TOKEN mysql ...` → `${TOKEN}`
- `exec` runs before Chrome is launched — useful for setup (creating users, cleaning DB)

### Examples

Login flow:

```
# Login with valid credentials

open http://localhost:9092/admin/login
fill Email admin@demo.com
fill Password 123
click Login

assert url /admin/dashboard
assert text Dashboard
```

Failed login:

```
# Login with wrong password shows error

open http://localhost:9092/admin/login
fill Email admin@demo.com
fill Password wrong
click Login

assert url /admin/login
assert text "Invalid credentials"
assert gone Dashboard
```

Form submission:

```
# Create a new user

open http://localhost:9092/admin/users
click "New user"

assert url /admin/users/new
fill "First name" Santiago
fill "Last name" Test
fill Email test@example.com
select Role Admin
click Save

assert text "User created"
```

JavaScript assertion:

```
# Page has no console errors

open http://localhost:9092/admin/dashboard
assert js "document.querySelectorAll('.error').length === 0"
assert js "document.title.includes('Dashboard')"
```

### Output

```
PASS  login_ok.bro — Login with valid credentials (1.2s)
PASS  create_user.bro — Create a new user (3.4s)
FAIL  delete_record.bro:12 — assert text: "Record deleted" not found

3 tests, 2 passed, 1 failed (4.8s)
```

---

## Multiple Sessions

Each `bro open` launches a separate Chrome instance with its own port and profile directory. Sessions are fully isolated — no shared cookies, localStorage, or state.

```bash
# Terminal 1
PORT=$(bro open http://localhost:9092/admin/login)
bro --port $PORT fill Email admin@demo.com
bro --port $PORT click Login
bro --port $PORT close

# Terminal 2 (runs in parallel, completely isolated)
PORT=$(bro open http://localhost:9092/admin/login)
bro --port $PORT fill Email other@demo.com
bro --port $PORT click Login
bro --port $PORT close
```

## How It Works

- `bro open` launches Chrome with `--remote-debugging-port` and a unique `--user-data-dir` per port
- All commands connect via WebSocket to Chrome's DevTools Protocol, act, and exit
- Element lookup uses Chrome's **accessibility tree** by default — reliable with any framework
- `--css` and `--id` flags use DOM selectors as an alternative for elements not in the AX tree
- Interactive elements (buttons, links) are prioritized over static text when names match
- Zero-width characters in element names/placeholders are stripped for robust matching
- `bro js` uses `Runtime.evaluate` with `awaitPromise` so Promises resolve automatically
- `bro close` sends a close command via CDP to cleanly shut down the Chrome instance

Built on [Rod](https://github.com/go-rod/rod).
