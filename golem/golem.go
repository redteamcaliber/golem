package golem

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/adblock"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/webkit"
)

// scrollbarHideCSS is the CSS to hide the scroll bars for webkit.
const scrollbarHideCSS = `
html::-webkit-scrollbar {
	height: 0 !important;
	width:  0 !important;
}`

// injectedHTMLCSS is CSS governing the display of specifically injected
// html code from golem.
const injectedHTMLCSS = `
.__golem-hint {
	padding: 1px;
	border: 1px solid rgba(0, 0, 0, 0.7);
	background-color: rgba(255, 255, 255, 0.7);
	position: absolute;
	font: bold 9pt monospace;
	color: rgba(0, 0, 0, 0.7);
	z-index: 100000;
}
.__golem-highlight {
	background-color: rgba(255, 255, 0, 0.5);
}
.__golem-hide {
	display: none;
}
`

// A uriEntry is a single uri with a title.
type uriEntry struct {
	uri   string
	title string
}

// Golem is golem's main instance.
type Golem struct {
	*globalCfg
	*RPCSession
	windows            []*Window
	webViews           map[uint64]*webView
	userContentManager *webkit.UserContentManager
	closeChan          chan<- *Window
	Quit               chan bool
	wMutex             *sync.Mutex
	rawBindings        []cmd.RawBinding

	DefaultSettings *webkit.Settings
	files           *files
	extenDir        string

	webViewCache          []*webView
	webViewCacheFree      chan bool
	webViewCacheClipboard string
	webViewCachePrimary   string

	historyMutex *sync.Mutex
	history      []uriEntry

	silentDownloads map[uintptr]bool

	adblocker *adblock.Blocker
}

// New creates a new instance of golem.
func New(session *RPCSession, profile string) (*Golem, error) {
	ucm, err := webkit.NewUserContentManager()
	if err != nil {
		return nil, err
	}
	css, err := webkit.NewUserStyleSheet(
		scrollbarHideCSS,
		webkit.UserContentInjectTopFrame,
		webkit.UserStyleLevelUser,
		nil,
		nil)
	if err != nil {
		return nil, err
	}
	ucm.AddStyleSheet(css)
	css, err = webkit.NewUserStyleSheet(
		injectedHTMLCSS,
		webkit.UserContentInjectAllFrames,
		webkit.UserStyleLevelAuthor,
		nil,
		nil)
	if err != nil {
		return nil, err
	}
	ucm.AddStyleSheet(css)

	closeChan := make(chan *Window)
	quitChan := make(chan bool)

	g := &Golem{
		defaultCfg,
		session,
		make([]*Window, 0, 10),
		make(map[uint64]*webView, 500),
		ucm,
		closeChan,
		quitChan,
		new(sync.Mutex),
		make([]cmd.RawBinding, 0, 100),
		webkit.NewSettings(),
		nil,
		"",
		make([]*webView, 0),
		make(chan bool, 1),
		"",
		"",
		new(sync.Mutex),
		make([]uriEntry, 0, defaultCfg.maxHistLen),
		make(map[uintptr]bool, 10),
		nil,
	}

	session.golem = g

	g.profile = profile

	g.files, err = g.newFiles()
	if err != nil {
		return nil, err
	}
	err = g.loadHistory()
	if err != nil {
		return nil, err
	}

	g.adblocker = adblock.NewBlocker(g.files.filterlistDir)

	g.webkitInit()

	for _, rcfile := range g.files.rcFiles() {
		err := g.useRcFile(rcfile)
		if err != nil {
			return nil, err
		}
	}

	return g, nil
}

// loadHistory loads an existing histfile.
func (g *Golem) loadHistory() error {
	data, err := ioutil.ReadFile(g.files.histfile)
	if os.IsNotExist(err) {
		// No history to load. Nothing to do.
	} else if err != nil {
		return err
	} else {
		histStrs := strings.Split(string(data), "\n")
		for _, str := range histStrs {
			split := strings.SplitN(str, "\t", 2)
			var uri, title string
			if len(split) != 2 {
				uri = split[0]
				title = ""
			} else {
				uri = split[0]
				title = split[1]
			}
			g.history = append(g.history, uriEntry{uri, title})
		}
	}
	return nil
}

// clipboardChanged checks if the contents of the clipboard has changed since
// the last write to the webViewCache.
func (g *Golem) clipboardChanged() bool {
	clip, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		(*Window)(nil).logErrorf("Failed to retrieve clipboard: %v", err)
		return false
	}
	text, err := clip.WaitForText()
	if err != nil {
		(*Window)(nil).logErrorf("Failed to retrieve clipboard text: %v", err)
		return false
	}
	return text != g.webViewCacheClipboard
}

// primaryChanged checks if the contents of the clipboard has changed since
// the last write to the webViewCache.
func (g *Golem) primaryChanged() bool {
	clip, err := gtk.ClipboardGet(gdk.SELECTION_PRIMARY)
	if err != nil {
		(*Window)(nil).logErrorf(
			"Failed to retrieve primary selection: %v", err)
		return false
	}
	text, err := clip.WaitForText()
	if err != nil {
		(*Window)(nil).logErrorf(
			"Failed to retrieve primary selection text: %v", err)
		return false
	}
	return text != g.webViewCachePrimary
}

// cutWebViews moves the supplied web views to an internal buffer, and keeps
// them there for at most 1 minute.
func (g *Golem) cutWebViews(wvs []*webView) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	g.webViewCacheFree <- true
	g.webViewCacheFree = make(chan bool, 1)
	g.webViewCache = wvs
	cp := make([]*webView, len(g.webViewCache))
	copy(cp, g.webViewCache)
	clipboards := []*string{&g.webViewCacheClipboard, &g.webViewCachePrimary}
	for i, sel := range []gdk.Atom{
		gdk.SELECTION_CLIPBOARD,
		gdk.SELECTION_PRIMARY} {

		clip, err := gtk.ClipboardGet(sel)
		if err != nil {
			(*Window)(nil).logErrorf("Failed to retrieve selection: %v", err)
			continue
		}
		text, err := clip.WaitForText()
		if err != nil {
			text = ""
		}
		*(clipboards[i]) = text
	}
	go func() {
		select {
		case <-time.After(time.Minute):
			g.wMutex.Lock()
			if sliceEquals(g.webViewCache, wvs) {
				g.webViewCache = make([]*webView, 0)
			}
			g.wMutex.Unlock()
		case free := <-g.webViewCacheFree:
			if !free {
				return
			}
		}
		for _, wv := range cp {
			wv.close()
		}
	}()
}

// pasteWebViews retrieves the contents of the web view cache and resets it
// safely.
func (g *Golem) pasteWebViews() (wvs []*webView) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	g.webViewCacheFree <- false
	g.webViewCacheFree = make(chan bool, 1)
	wvs = g.webViewCache
	g.webViewCache = make([]*webView, 0)
	return wvs
}

// useRcFile reads and executes an rc file.
func (g *Golem) useRcFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		runCmd(nil, g, scanner.Text())
	}
	return nil
}

// bind creates a new key binding.
func (g *Golem) bind(from string, to string) {
	// We check if the key has been bound before. If so, we replace the
	// binding.
	index := -1
	for i, b := range g.rawBindings {
		if from == b.From {
			index = i
			break
		}
	}

	g.wMutex.Lock()
	if index != -1 {
		g.rawBindings[index] = cmd.RawBinding{from, to}
	} else {
		g.rawBindings = append(g.rawBindings, cmd.RawBinding{from, to})
	}
	g.wMutex.Unlock()

	for _, w := range g.windows {
		w.rebuildBindings()
	}
}

// quickmark adds a quickmark to golem.
func (g *Golem) quickmark(from string, title string, uri string) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	g.quickmarks[from] = uriEntry{uri, title}
	g.hasQuickmark[uri] = true

	for _, w := range g.windows {
		w.rebuildQuickmarks()
	}
}

// Close closes golem.
func (g *Golem) Close() {
	for _, w := range g.windows {
		w.Close()
	}
}

// closeWindow updates bookkeeping after a window was closed.
func (g *Golem) closeWindow(w *Window) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	// w points to the window which was closed. It will be removed
	// from golems window list, and in doing so will be deferenced.
	var i int
	found := false
	for i = range g.windows {
		if g.windows[i] == w {
			found = true
			break
		}
	}
	if !found {
		(*Window)(nil).logError("Close signal recieved for non-existant " +
			"window. Dropping.")
	}

	// Delete item at index i from the slice.
	l := len(g.windows)
	copy(g.windows[i:l-1], g.windows[i+1:l])
	g.windows[l-1] = nil
	g.windows = g.windows[0 : l-1]

	// If no windows are left, golem exits
	if len(g.windows) == 0 {
		gtk.MainQuit()
		g.Quit <- true
	}
}

// addDownload adds a new download to the tracked downloads.
func (g *Golem) addDownload(d *webkit.Download) {
}

// updateHistory updates the history file. With a newly visited uri and title.
func (g *Golem) updateHistory(uri, title string) {
	if g.maxHistLen == 0 || uri == "" {
		return
	}
	g.historyMutex.Lock()
	defer g.historyMutex.Unlock()
	// Check if uri is alreay in the history. If so, move to the end, and
	// update title.
	var i int
	for i = 0; i < len(g.history); i++ {
		if g.history[i].uri == uri {
			break
		}
	}
	if i != len(g.history) {
		// Update title and move to end.
		hist := g.history[i]
		hist.title = title
		copy(g.history[i:len(g.history)-1], g.history[i+1:])
		g.history[len(g.history)-1] = hist
	} else {
		if uint(len(g.history)) == g.maxHistLen {
			g.history = g.history[1:]
		}
		g.history = append(g.history, uriEntry{uri, title})
	}
	// Write hist file.
	strHist := make([]string, len(g.history))
	for i, hist := range g.history {
		strHist[i] = fmt.Sprintf("%s\t%s", hist.uri, hist.title)
	}
	err := ioutil.WriteFile(
		g.files.histfile,
		[]byte(strings.Join(strHist, "\n")),
		0600)
	if err != nil {
		(*Window)(nil).logErrorf("Failed to write history file: %v", err)
	}
}
