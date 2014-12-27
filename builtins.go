package main

import (
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
)

// builtinsfor retrieves the builtin functions bound to a specific window.
func builtinsFor(w *window) cmd.Builtins {
	return cmd.Builtins{
		"commandMode":    w.builtinCommandMode,
		"editURI":        w.builtinEditURI,
		"goBack":         w.builtinGoBack,
		"goForward":      w.builtinGoForward,
		"insertMode":     w.builtinInsertMode,
		"nop":            w.builtinNop,
		"open":           w.builtinOpen,
		"panic":          w.builtinPanic,
		"reload":         w.builtinReload,
		"runCmd":         w.builtinRunCmd,
		"scrollDown":     w.builtinScrollDown,
		"scrollLeft":     w.builtinScrollLeft,
		"scrollRight":    w.builtinScrollRight,
		"scrollToBottom": w.builtinScrollToBottom,
		"scrollToTop":    w.builtinScrollToTop,
		"scrollUp":       w.builtinScrollUp,
		"tabClose":       w.builtinTabClose,
		"tabEditURI":     w.builtinTabEditURI,
		"tabNext":        w.builtinTabNext,
		"tabOpen":        w.builtinTabOpen,
		"tabPrev":        w.builtinTabPrev,
		"windowEditURI":  w.builtinWindowEditURI,
		"windowOpen":     w.builtinWindowOpen,
	}
}

// builtinCommandMode initiates command mode.
func (w *window) builtinCommandMode(_ ...interface{}) {
	w.setState(cmd.NewCommandLineMode(w.State, w.runCmd))
}

// builtinEditURI initiates command mode with the open command primed for
// the current URI.
func (w *window) builtinEditURI(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		fmt.Sprintf("open %v", w.WebView.GetURI()),
		w.runCmd))
}

// builtinGoBack goes one step back in browser history.
func (w *window) builtinGoBack(_ ...interface{}) {
	w.WebView.GoBack()
}

// builtinGoForward goes one step forward in browser history.
func (w *window) builtinGoForward(_ ...interface{}) {
	w.WebView.GoForward()
}

// builtinInsertMode initiates insert mode.
func (w *window) builtinInsertMode(_ ...interface{}) {
	w.setState(cmd.NewInsertMode(w.State))
}

// builtinNop does nothing. It is occasionally useful as a binding.
func (w *window) builtinNop(_ ...interface{}) {}

// builtinOpen initiates command mode, primed with an open command.
func (w *window) builtinOpen(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, "open ", w.runCmd))
}

func (w *window) builtinPanic(_ ...interface{}) {
	panic("Builtin 'panic' called.")
}

// builtinReload reloads the current page.
func (w *window) builtinReload(_ ...interface{}) {
	w.WebView.Reload()
}

// builtinRunCmd runs a command with its first argument as a string.
func (w *window) builtinRunCmd(args ...interface{}) {
	if len(args) < 1 {
		log.Printf("Failed to execute builtin 'runCmd': Not enough arguments")
		return
	}
	cmd, ok := args[0].(string)
	if !ok {
		log.Printf(
			"Invalid type for argument for builtin 'runCmd': %T",
			args[0])
	}
	w.runCmd(cmd)
}

// builtinScrollDown scrolls down.
func (w *window) builtinScrollDown(_ ...interface{}) {
	w.scrollDelta(w.parent.scrollDelta, true)
}

// builtinScrollLeft scrolls left.
func (w *window) builtinScrollLeft(_ ...interface{}) {
	w.scrollDelta(-w.parent.scrollDelta, false)
}

// builtinScrollRight scrolls right.
func (w *window) builtinScrollRight(_ ...interface{}) {
	w.scrollDelta(w.parent.scrollDelta, false)
}

// builtinScrollToBottom scrolls to the bottom of the page.
func (w *window) builtinScrollToBottom(_ ...interface{}) {
	ext := w.getWebView()
	height, err := ext.getScrollHeight()
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
	err = ext.setScrollTop(height)
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
}

// builtinScrollTotop scrolls to the top of the page.
func (w *window) builtinScrollToTop(_ ...interface{}) {
	err := w.getWebView().setScrollTop(0)
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
}

// builtinScrollUp scrolls up.
func (w *window) builtinScrollUp(_ ...interface{}) {
	w.scrollDelta(-w.parent.scrollDelta, true)
}

// builtinTabClose closes the current tab.
func (w *window) builtinTabClose(_ ...interface{}) {
	w.tabClose()
}

// builtinTabEditURI initiates command mode with a tabopen command primed for
// the current URI.
func (w *window) builtinTabEditURI(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		fmt.Sprintf("tabopen %v", w.GetURI()),
		w.runCmd))
}

// builtinTabNext goes to the next tab.
func (w *window) builtinTabNext(_ ...interface{}) {
	w.tabNext()
}

// builtinTabOpen initiates command mode primed with a tabopen command.
func (w *window) builtinTabOpen(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, "tabopen ", w.runCmd))
}

// builtinTabPrev goes to the previous tab.
func (w *window) builtinTabPrev(_ ...interface{}) {
	w.tabPrev()
}

// builtinWindowEditURI initiates command mode with a winopen command primed
// for the current URI.
func (w *window) builtinWindowEditURI(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		fmt.Sprintf("winopen %v", w.GetURI()),
		w.runCmd))
}

// builtinWindowOpen initiates command mode primed with a winopen command.
func (w *window) builtinWindowOpen(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, "winopen ", w.runCmd))
}

// scrollDelta scrolls a given amount of pixes either vertically or
// horizontally.
func (w *window) scrollDelta(delta int, vertical bool) {
	var curr int64
	var err error
	wv := w.getWebView()
	if vertical {
		curr, err = wv.getScrollTop()
	} else {
		curr, err = wv.getScrollLeft()
	}
	if err != nil {
		log.Printf("Error scrolling: %v", err)
		return
	}
	curr += int64(delta)
	if vertical {
		err = wv.setScrollTop(curr)
	} else {
		err = wv.setScrollLeft(curr)
	}
	if err != nil {
		log.Printf("Error scrolling: %v", err)
		return
	}
}
