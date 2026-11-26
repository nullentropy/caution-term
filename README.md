# [Caution Term](https://nullentropy.github.io/caution-term/)

Caution Term is a small terminal for macOS. It runs your login shell with
tabs, scrollback and mouse selection, and can put a shader over the whole
window. It is built on [caution](https://github.com/nullentropy/caution).

It is also a Go package for apps that run a terminal program in a window.
[Caution Emacs](https://github.com/nullentropy/caution-emacs) is built on it.

## Requirements

- macOS 13 or newer.
- Optional: Fira Code Nerd Font Mono in `~/Library/Fonts`. Without it the app
  uses caution's built-in monospace font.

To build from source:

- Go 1.26 or newer.
- The Xcode Command Line Tools, for the C compiler, `codesign` and `lipo`.

## Install

```
curl -fsSL https://raw.githubusercontent.com/nullentropy/caution-term/main/install.sh | sh
```

This downloads the latest release into `/Applications`, replacing an existing
`Caution Term.app` and quitting it first if it is running. Set
`CAUTION_TERM_DEST` to install in another folder.

Or install from a release by hand. Unzip
`CautionTerm-<version>-macos-universal.zip` and move `Caution Term.app` to
Applications.

The app is ad hoc signed, not notarized. A copy downloaded in a browser is
quarantined and macOS will refuse to open it. Either try to open it once, then
go to System Settings > Privacy & Security and choose Open Anyway, or clear
the flag:

```
xattr -dr com.apple.quarantine "/Applications/Caution Term.app"
```

## Run from source

```
go run .                  # your login shell
go run . -look CRT        # with a look
go run . -- htop          # any other program
go run . -addr :8790      # serve to a browser instead of opening a window
```

Other flags: `-font` picks the font files and `-shot file.png` renders one
frame offscreen and quits.

## Build the app

```
scripts/app.sh
```

This writes `dist/Caution Term.app`, ad hoc signed, for your Mac's
architecture. Drag it to Applications. Opened from Finder or the Dock, it takes
`PATH` from your login shell and starts in your home folder.

`go run ./icon` redraws `icon.png` and `icon.svg`.

## Make a release

Push a tag such as `v1.2.3`. `.github/workflows/release.yml` builds the app on
a macOS runner and attaches these files to the release for that tag, creating
the release if it does not exist:

- `CautionTerm-<version>-macos-universal.zip`
- `CautionTerm-macos-universal.zip`, the one `install.sh` downloads
- `SHA256SUMS`

To build the zips by hand:

```
scripts/release.sh          # version from the tag on HEAD
scripts/release.sh 1.2.3    # or pass the version
```

The working tree must be clean. Without an argument, HEAD must have a tag; a
leading `v` is dropped. The script builds arm64 and amd64 binaries, joins them
into one universal app, sets the version in its `Info.plist`, signs it ad hoc
and writes the zips to `release/`.

## Using it

### Keyboard shortcuts

| Key | Action |
|---|---|
| `Cmd+N` | New window |
| `Cmd+T` | New tab |
| `Cmd+W` | Close tab |
| `Cmd+Shift+]` / `Cmd+Shift+[` | Next / previous tab |
| `Cmd+C` | Copy the selection |
| `Cmd+V` | Paste |
| `Cmd+K` | Clear scrollback |
| `Cmd+=` / `Cmd+-` / `Cmd+0` | Bigger / smaller / actual size text |
| `Cmd+]` / `Cmd+[` | Next / previous look |
| `Shift+Page Up` / `Shift+Page Down` | Scroll back / forward a page |
| `Shift+Home` / `Shift+End` | Scroll to the top / back to the prompt |

Every other key goes to the program. `Option` is Meta.

### Scrolling and selecting

The trackpad or wheel scrolls back through lines that have left the screen.
Typing jumps back to the prompt. In a full-screen program that does not use
the mouse, such as `less`, the wheel sends the arrow keys instead.

Drag to select, double-click to select a word, triple-click to select a line.
Typing clears the selection. A program that uses the mouse, such as htop or
Emacs, gets your clicks; hold `Shift` or `Option` to select text in it anyway.

Text a program sends to the terminal clipboard (OSC 52) also lands on the
macOS clipboard.

### Tabs and windows

Each tab runs its own shell. A tab closes when its shell exits, and the window
closes with its last tab. A new window is a new copy of the app.

### Looks

View > Look puts a shader over the whole window: CRT, Green Phosphor, Amber
Phosphor, Bloom, Chromatic, Vaporwave, Underwater, Matrix Rain, Thermal or
Glitch. The animated ones redraw every frame, so they use more power than the
still ones.

### Where things are kept

Text size, the current look and the window's size are kept in
`~/Library/Application Support/Caution Term/settings.json`.

## Use the package

The package is `github.com/nullentropy/caution-term/term`.

```go
w := term.NewWindow(session, term.WindowOptions{Fonts: term.LoadFonts(term.DefaultFonts)})
t := w.Open(term.Options{Argv: []string{"emacs", "-nw"}})
session.SetMenu(term.ViewMenu(w, nil))
return w.Node()
```

A `Window` is the title strip, the tabs, the text size and the look. Each
`Open` adds a tab running a program. A `Term` can paste, copy its selection,
clear its scrollback and restart its program. `term.LoadSettings` keeps the
text size, look and window size between launches, and `term.PrepareBundle`
takes `PATH` from the login shell when the app is opened from Finder.

`go test ./term` runs a program through a real caution session and checks
scrollback, word and drag selection, and the copied text.
