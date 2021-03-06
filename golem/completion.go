package golem

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
	"github.com/tkerber/golem/gtk"
	"github.com/tkerber/golem/webkit"
)

var trailingWhitespaceRegex = regexp.MustCompile(`.*\s$`)

// completeState starts completing a state.
func (w *Window) completeState(
	s cmd.State,
	firstFunc func(bool),
	compStates *[]cmd.State) (cancel func()) {

	var strs []string
	updated := false
	i := 0
	update := func(done bool) {
		if i%100 == 0 || done {
			w.Window.CompletionBar.UpdateCompletions(strs)
		}
		i++
		if !updated {
			firstFunc(!done)
			updated = true
			gtk.GlibMainContextInvoke(
				w.Window.CompletionBar.Container.Show)
		}
	}
	cancelled := false
	go w.parent.complete(s, &cancelled, update, compStates, &strs)
	cancel = func() {
		cancelled = true
		w.Window.CompletionBar.UpdateCompletions(nil)
		w.Window.CompletionBar.UpdateAt(0)
		gtk.GlibMainContextInvoke(
			w.Window.CompletionBar.Container.Hide)
	}
	return cancel
}

// complete retrieves the possible completions for a state and started them
// in a slice at the passed pointer.
//
// Complete is intended to be run with a go statement:
//	go complete(s, cancelCompletion, ptr)
//
// Sending to the cancel channel terminates execution of the function (at
// pre-set intervals). It is recommended to buffer the cancel channel and
// limit to sending one item, as it isn't guaranteed to be read.
//
// Passing nil for ptr is a fatal error.
func (g *Golem) complete(
	s cmd.State,
	cancelled *bool,
	update func(bool),
	compStates *[]cmd.State,
	compStrings *[]string) {

	switch s := s.(type) {
	case *cmd.NormalMode:
		g.completeNormalMode(s, cancelled, update, compStates, compStrings)
	case *cmd.CommandLineMode:
		f := g.completeCommandLineMode(s)
		for {
			s, str, ok := f()
			if !ok {
				break
			}
			if *cancelled {
				return
			}
			*compStates = append(*compStates, s)
			*compStrings = append(*compStrings, str)
			update(false)
		}
		update(true)
	default:
		return
	}
}

// completeCommandLineMode completes a command line mode state.
func (g *Golem) completeCommandLineMode(
	s *cmd.CommandLineMode) func() (cmd.State, string, bool) {

	// Only the keys before the cursor are taken into account.
	keyStr := cmd.KeysStringSelective(s.CurrentKeys[:s.CursorPos], false)
	switch s.Substate {
	case states.CommandLineSubstateCommand:
		return g.completionWrapCommandLine(g.completeCommand(keyStr), s)
	default:
		return func() (cmd.State, string, bool) {
			return nil, "", false
		}
	}
}

// completionWrapCommandLine wraps a command string generator into a
// completion state generator.
func (g *Golem) completionWrapCommandLine(
	f func() (string, string, bool),
	s *cmd.CommandLineMode) func() (cmd.State, string, bool) {

	return func() (cmd.State, string, bool) {
		keyStr, desc, ok := f()
		if !ok {
			return nil, "", false
		}
		keys := cmd.ParseKeys(keyStr)
		return &cmd.CommandLineMode{
				s.StateIndependant,
				s.Substate,
				keys,
				len(keys),
				s.CursorHome,
				0,
				s.Finalizer,
			},
			desc,
			true
	}
}

// completeCommand Completes a command state.
func (g *Golem) completeCommand(command string) func() (string, string, bool) {
	parts, err := shellwords.Parse(command)
	if err != nil {
		return func() (string, string, bool) {
			return "", "", false
		}
	}
	// If we have a trailing whitespace, we add an empty part. This is so that
	// for "open " e.g. a uri will be completed and not a command.
	if trailingWhitespaceRegex.MatchString(command) {
		parts = append(parts, "")
	}
	if len(parts) <= 1 {
		// Silly name. But we actually complete the "command" part of the
		// command here.
		return g.completeCommandCommand(command)
	}
	switch parts[0] {
	case "aqm", "addquickmark", "qm", "quickmark":
		// complete url from 2nd parameter onwards.
		return g.completeURI(parts, 2)
	case "o", "open",
		"t", "topen", "tabopen", "newtab",
		"bg", "bgopen", "backgroundopen",
		"w", "wopen", "winopen", "windowopen":
		// complete url from 1st parameter onwards.
		return g.completeURI(parts, 1)
	case "bind":
		// complete builtin/command from 2nd paramter onwards.
		return g.completeBinding(parts)
	case "set":
		// complete setting name from 1st parameter onwards.
		return g.completeOptionSet(parts)
	case "rmqm", "removerequickmark":
		// complete quickmark
		return g.completeQuickmark(parts)
	case "q", "quit", "qall", "quitall":
		fallthrough
	default:
		return func() (string, string, bool) {
			return "", "", false
		}
	}
}

// completeOptionSet complets an option for the "set" command.
func (g *Golem) completeOptionSet(
	parts []string) func() (string, string, bool) {

	if len(parts) != 2 {
		return func() (string, string, bool) {
			return "", "", false
		}
	}
	i := -1
	return func() (string, string, bool) {
		for {
			i++
			if i >= len(webkit.SettingNames) {
				return "", "", false
			}
			setting := webkit.SettingNames[i]
			if strings.HasPrefix("w:"+setting, parts[1]) ||
				strings.HasPrefix("webkit:"+setting, parts[1]) {

				t, _ := webkit.GetSettingsType(setting)
				return parts[0] + " webkit:" + setting,
					fmt.Sprintf(
						"%s\t%v\tWebkit",
						setting,
						t),
					true
			}
		}
	}
}

// completeQuickmarks completes a quickmark argument.
func (g *Golem) completeQuickmark(
	parts []string) func() (string, string, bool) {

	qml := make([]string, 0, len(g.quickmarks))
	for qm := range g.quickmarks {
		qml = append(qml, qm)
	}
	i := -1
	return func() (string, string, bool) {
		for {
			i++
			if i >= len(qml) {
				return "", "", false
			} else if strings.HasPrefix(qml[i], parts[1]) {
				return qml[i],
					fmt.Sprintf("%s\t%s\t%s",
						qml[i],
						g.quickmarks[qml[i]].uri,
						g.quickmarks[qml[i]].title),
					true
			}
		}
	}
}

// completeBinding completes a binding argument.
func (g *Golem) completeBinding(
	parts []string) func() (string, string, bool) {

	opt := ""
	if len(parts) == 3 {
		opt = parts[2]
	} else {
		return func() (string, string, bool) {
			return "", "", false
		}
	}
	i := -1
	return func() (string, string, bool) {
		for {
			i++
			if i > len(commandNames)+len(builtinNames) {
				return "", "", false
			} else if i < len(builtinNames) {
				if strings.HasPrefix("builtin:"+builtinNames[i], opt) ||
					strings.HasPrefix("b:"+builtinNames[i], opt) {

					return parts[0] + "builtin:" + builtinNames[i],
						fmt.Sprintf("%s\tbuiltin", builtinNames[i]),
						true
				}
			} else {
				j := i - len(builtinNames)
				if strings.HasPrefix("command:"+commandNames[j], opt) ||
					strings.HasPrefix("cmd:"+commandNames[j], opt) ||
					strings.HasPrefix("c:"+commandNames[j], opt) {

					return parts[0] + "cmd:" + commandNames[j],
						fmt.Sprintf("%s\tcommand", commandNames[j]),
						true
				}
			}
		}
	}
}

// completeURI completes a URI argument.
func (g *Golem) completeURI(
	parts []string,
	startFrom int) func() (string, string, bool) {

	uriparts := parts[startFrom:]
	stage := 0
	qmArr := make([]uriEntry, len(g.quickmarks))
	i := 0
	for _, s := range g.quickmarks {
		qmArr[i] = s
		i++
	}
	i = -1
	return func() (string, string, bool) {
		var uri string
		// Where the uri came from (quickmarks, bookmarks, history)
		var uriType string
		var title string
	outer:
		for {
			switch stage {
			// complete quickmarks
			case 0:
				i++
				if i >= len(qmArr) {
					stage++
					i = -1
					continue
				}
				for _, part := range uriparts {
					if !strings.Contains(qmArr[i].uri, part) {
						continue outer
					}
				}
				uri = qmArr[i].uri
				title = qmArr[i].title
				uriType = "Quickmark"
				break outer
			// complete bookmarks
			case 1:
				i++
				if i >= len(g.bookmarks) {
					stage++
					i = -1
					continue
				}
				for _, part := range uriparts {
					if !strings.Contains(g.bookmarks[i].uri, part) {
						continue outer
					}
				}
				uri = g.bookmarks[i].uri
				title = g.bookmarks[i].title
				uriType = "Bookmark"
				break outer
			// complete history
			case 2:
				i++
				if i >= len(g.history) {
					stage++
					continue
				}
				item := g.history[len(g.history)-1-i]
				for _, part := range uriparts {
					if !strings.Contains(item.uri, part) &&
						!strings.Contains(item.title, part) {

						continue outer
					}
				}
				uri = item.uri
				title = item.title
				uriType = "History"
				break outer
			// end iteration
			default:
				return "", "", false
			}
		}
		// Won't always cleanly work. But it doesn't have to.
		return strings.Join(parts[:startFrom], " ") + " " + uri,
			fmt.Sprintf("%s\t%s\t%s", uri, title, uriType),
			true
	}
}

// stringCompleteAgainstList returns a function iterating over possible
// completions for the given string, amount the given list.
func stringCompleteAgainstList(
	str string,
	arr []string) func() (string, string, bool) {

	i := 0
	return func() (string, string, bool) {
		for i < len(arr) {
			if strings.HasPrefix(arr[i], str) {
				i++
				return arr[i-1], arr[i-1], true
			}
			i++
		}
		return "", "", false
	}
}

// completeCommandCommand completes the actual command of a command mode.
func (g *Golem) completeCommandCommand(
	cmd string) func() (string, string, bool) {

	commandNames := make([]string, len(commands))
	i := 0
	for command := range commands {
		commandNames[i] = command
		i++
	}
	return stringCompleteAgainstList(cmd, commandNames)
}

// completeNormalMode completes a normal mode state.
func (g *Golem) completeNormalMode(
	s *cmd.NormalMode,
	cancelled *bool,
	update func(bool),
	compStates *[]cmd.State,
	compStrings *[]string) {

outer:
	for b := range s.CurrentTree.IterLeaves() {
		if *cancelled {
			return
		}
		// We can't complete virtual keys.
		for _, key := range b.From {
			if _, ok := key.(cmd.VirtualKey); ok {
				continue outer
			}
		}
		// Get the new tree
		t := s.CurrentTree
		for _, k := range b.From {
			t = t.Subtrees[k]
		}
		var str string
		keysStr := cmd.KeysString(b.From)
		switch s.Substate {
		case states.NormalSubstateNormal:
			str = fmt.Sprintf("%s\t%s\t%s", keysStr, b.Name, b.Desc)
		case states.NormalSubstateQuickmark,
			states.NormalSubstateQuickmarkTab,
			states.NormalSubstateQuickmarkWindow,
			states.NormalSubstateQuickmarksRapid:

			str = fmt.Sprintf("%s\t%s\t%s",
				keysStr,
				g.quickmarks[keysStr].uri,
				g.quickmarks[keysStr].title)
		}
		*compStates = append(*compStates, s.PredictState(b.From[len(s.CurrentKeys):]))
		*compStrings = append(*compStrings, str)
		update(false)
	}
	update(true)
}
