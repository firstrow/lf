package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func (e *SetExpr) eval(app *App, args []string) {
	switch e.opt {
	case "hidden":
		gOpts.hidden = true
		app.nav.renew(app.nav.height)
	case "nohidden":
		gOpts.hidden = false
		app.nav.renew(app.nav.height)
	case "hidden!":
		gOpts.hidden = !gOpts.hidden
		app.nav.renew(app.nav.height)
	case "preview":
		gOpts.preview = true
	case "nopreview":
		gOpts.preview = false
	case "preview!":
		gOpts.preview = !gOpts.preview
	case "scrolloff":
		n, err := strconv.Atoi(e.val)
		if err != nil {
			app.ui.message = err.Error()
			log.Print(err)
			return
		}
		if n < 0 {
			msg := "scrolloff should be a non-negative number"
			app.ui.message = msg
			log.Print(msg)
			return
		}
		max := app.ui.wins[0].h/2 - 1
		if n > max {
			// TODO: stay at the same row while up/down in the middle
			n = max
		}
		gOpts.scrolloff = n
	case "tabstop":
		n, err := strconv.Atoi(e.val)
		if err != nil {
			app.ui.message = err.Error()
			log.Print(err)
			return
		}
		if n <= 0 {
			msg := "tabstop should be a positive number"
			app.ui.message = msg
			log.Print(msg)
			return
		}
		gOpts.tabstop = n
	case "ifs":
		gOpts.ifs = e.val
	case "showinfo":
		if e.val != "none" && e.val != "size" && e.val != "time" {
			msg := "showinfo should either be 'none', 'size' or 'time'"
			app.ui.message = msg
			log.Print(msg)
			return
		}
		gOpts.showinfo = e.val
	case "sortby":
		if e.val != "name" && e.val != "size" && e.val != "time" {
			msg := "sortby should either be 'name', 'size' or 'time'"
			app.ui.message = msg
			log.Print(msg)
			return
		}
		gOpts.sortby = e.val
		app.nav.renew(app.nav.height)
	case "opener":
		gOpts.opener = e.val
	case "ratios":
		toks := strings.Split(e.val, ":")
		var rats []int
		for _, s := range toks {
			i, err := strconv.Atoi(s)
			if err != nil {
				log.Print(err)
				return
			}
			rats = append(rats, i)
		}
		gOpts.ratios = rats
		app.ui = newUI()
	default:
		msg := fmt.Sprintf("unknown option: %s", e.opt)
		app.ui.message = msg
		log.Print(msg)
	}
}

func (e *MapExpr) eval(app *App, args []string) {
	gOpts.keys[e.keys] = e.expr
}

func (e *CmdExpr) eval(app *App, args []string) {
	gOpts.cmds[e.name] = e.expr
}

func (e *CallExpr) eval(app *App, args []string) {
	// TODO: check for extra toks in each case
	switch e.name {
	case "quit":
		gExitFlag = true
	case "echo":
		app.ui.message = strings.Join(e.args, " ")
	case "down":
		app.nav.down()
		app.ui.echoFileInfo(app.nav)
	case "up":
		app.nav.up()
		app.ui.echoFileInfo(app.nav)
	case "updir":
		err := app.nav.updir()
		if err != nil {
			app.ui.message = err.Error()
			log.Print(err)
			return
		}
		app.ui.echoFileInfo(app.nav)
	case "open":
		dir := app.nav.currDir()

		if len(dir.fi) == 0 {
			return
		}

		path := app.nav.currPath()

		f, err := os.Stat(path)
		if err != nil {
			app.ui.message = err.Error()
			log.Print(err)
			return
		}

		if f.IsDir() {
			err := app.nav.open()
			if err != nil {
				app.ui.message = err.Error()
				log.Print(err)
				return
			}
			app.ui.echoFileInfo(app.nav)
			return
		}

		if gSelectionPath != "" {
			out, err := os.Create(gSelectionPath)
			if err != nil {
				msg := fmt.Sprintf("open: %s", err)
				app.ui.message = msg
				log.Print(msg)
				return
			}
			defer out.Close()

			if len(app.nav.marks) != 0 {
				marks := app.nav.currMarks()
				path = strings.Join(marks, "\n")
			}

			_, err = out.WriteString(path)
			if err != nil {
				msg := fmt.Sprintf("open: %s", err)
				app.ui.message = msg
				log.Print(msg)
				return
			}

			gExitFlag = true
			return
		}

		if len(app.nav.marks) == 0 {
			app.runShell(fmt.Sprintf("%s '%s'", gOpts.opener, path), nil, false, false)
		} else {
			s := gOpts.opener
			for m := range app.nav.marks {
				s += fmt.Sprintf(" '%s'", m)
			}
			app.runShell(s, nil, false, false)
		}
	case "bot":
		app.nav.bot()
		app.ui.echoFileInfo(app.nav)
	case "top":
		app.nav.top()
		app.ui.echoFileInfo(app.nav)
	case "cd":
		err := app.nav.cd(e.args[0])
		if err != nil {
			app.ui.message = err.Error()
			log.Print(err)
			return
		}
		app.ui.echoFileInfo(app.nav)
	case "read":
		s := app.ui.prompt(":")
		if len(s) == 0 {
			app.ui.echoFileInfo(app.nav)
			return
		}
		log.Printf("command: %s", s)
		p := newParser(strings.NewReader(s))
		for p.parse() {
			p.expr.eval(app, nil)
		}
		if p.err != nil {
			app.ui.message = p.err.Error()
			log.Print(p.err)
		}
	case "read-shell":
		s := app.ui.prompt("$")
		log.Printf("shell: %s", s)
		app.runShell(s, nil, false, false)
	case "read-shell-wait":
		s := app.ui.prompt("!")
		log.Printf("shell-wait: %s", s)
		app.runShell(s, nil, true, false)
	case "read-shell-async":
		s := app.ui.prompt("&")
		log.Printf("shell-async: %s", s)
		app.runShell(s, nil, false, true)
	case "search":
		s := app.ui.prompt("/")
		log.Printf("search: %s", s)
		app.ui.message = "sorry, search is not implemented yet!"
		// TODO: implement
	case "search-back":
		s := app.ui.prompt("?")
		log.Printf("search-back: %s", s)
		app.ui.message = "sorry, search-back is not implemented yet!"
		// TODO: implement
	case "toggle":
		app.nav.toggle()
	case "yank":
		err := app.nav.save(true)
		if err != nil {
			msg := fmt.Sprintf("yank: %s", err)
			app.ui.message = msg
			log.Printf(msg)
			return
		}
		app.nav.marks = make(map[string]bool)
	case "delete":
		err := app.nav.save(false)
		if err != nil {
			msg := fmt.Sprintf("delete: %s", err)
			app.ui.message = msg
			log.Printf(msg)
			return
		}
		app.nav.marks = make(map[string]bool)
	case "paste":
		err := app.nav.paste()
		if err != nil {
			msg := fmt.Sprintf("paste: %s", err)
			app.ui.message = msg
			log.Printf(msg)
			return
		}
		app.nav.renew(app.nav.height)
		app.nav.save(false)
		saveFiles(nil, false)
	case "redraw":
		app.ui.renew()
		app.nav.renew(app.ui.wins[0].h)
	default:
		cmd, ok := gOpts.cmds[e.name]
		if !ok {
			msg := fmt.Sprintf("command not found: %s", e.name)
			app.ui.message = msg
			log.Print(msg)
			return
		}
		cmd.eval(app, e.args)
	}
}

func (e *ExecExpr) eval(app *App, args []string) {
	switch e.pref {
	case "$":
		log.Printf("shell: %s -- %s", e, args)
		app.ui.clearMsg()
		app.runShell(e.expr, args, false, false)
		app.ui.echoFileInfo(app.nav)
	case "!":
		log.Printf("shell-wait: %s -- %s", e, args)
		app.runShell(e.expr, args, true, false)
	case "&":
		log.Printf("shell-async: %s -- %s", e, args)
		app.runShell(e.expr, args, false, true)
	case "/":
		log.Printf("search: %s -- %s", e, args)
		// TODO: implement
	case "?":
		log.Printf("search-back: %s -- %s", e, args)
		// TODO: implement
	default:
		log.Printf("unknown execution prefix: %q", e.pref)
	}
}

func (e *ListExpr) eval(app *App, args []string) {
	for _, expr := range e.exprs {
		expr.eval(app, nil)
	}
}
