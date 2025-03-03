package slog_align

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"zgo.at/jfmt"
	"zgo.at/termtext"
	"zgo.at/zli"
)

var colors = map[slog.Level]zli.Color{
	slog.LevelDebug: zli.Color256(250).Bg(),
	slog.LevelInfo:  zli.Color256(51).Bg(),
	slog.LevelWarn:  zli.Color256(214).Bg(),
	slog.LevelError: zli.Color256(210).Bg(),
}

// AlignedHandler is a handler for slog that prints values aligned.
type AlignedHandler struct {
	w        io.Writer
	g        []string
	attr     []slog.Attr
	replAttr func(groups []string, a slog.Attr) slog.Attr
	lvl      slog.Level
	timefmt  string
	indent   string
	root     string
	width    *int
	widthMu  *sync.Mutex
}

func NewAlignedHandler(w io.Writer, opt *slog.HandlerOptions) AlignedHandler {
	if opt == nil {
		opt = &slog.HandlerOptions{}
	}
	if opt.Level == nil {
		opt.Level = slog.LevelInfo
	}

	r := moduleRoot()
	if r != "" {
		r += string(os.PathSeparator)
	}

	h := AlignedHandler{
		w:        w,
		lvl:      opt.Level.Level(),
		replAttr: opt.ReplaceAttr,
		root:     r,
		widthMu:  new(sync.Mutex),
		width:    new(int),
	}

	if std, ok := w.(*os.File); ok && (std.Fd() == 1 || std.Fd() == 2) {
		w, _, _ := zli.TerminalSize(std.Fd())
		if w < 60 && w > 0 {
			w = 60
		}
		if w > 0 {
			h.setWidth(w)
		}

		if w > 0 {
			winChange := make(chan os.Signal, 1)
			signal.Notify(winChange, sigWinChange)
			go func() {
				for {
					<-winChange
					w, _, _ := zli.TerminalSize(std.Fd())
					if w < 60 && w > 0 {
						w = 60
					}
					if w > 0 {
						h.setWidth(w)
					}
				}
			}()
		}
	}

	h.SetTimeFormat("15:04")
	return h
}

// SetTimeFormat sets the timestsamp format.
//
// The default is 15:04. Use an empty string to disable outputting a timestamp.
func (h *AlignedHandler) SetTimeFormat(fmt string) {
	h.timefmt = fmt
	h.indent = strings.Repeat(" ", len(time.Now().Format(h.timefmt))+8)
	if fmt == "" {
		h.indent = "      "
	}
}

func (h *AlignedHandler) getWidth() int {
	h.widthMu.Lock()
	defer h.widthMu.Unlock()
	return *h.width
}

func (h *AlignedHandler) setWidth(w int) {
	h.widthMu.Lock()
	defer h.widthMu.Unlock()
	*h.width = w
}

// Enabled reports whether the handler handles records at the given level.
func (h AlignedHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return l >= h.lvl
}

var testTime *time.Time

// Handle the Record.
//
// Handle methods that produce output should observe the following rules:
//
//   - If r.Time is the zero time, ignore the time.
//   - If an Attr's key is the empty string, ignore the Attr.
func (h AlignedHandler) Handle(ctx context.Context, r slog.Record) error {
	if testTime != nil {
		r.Time = *testTime
	}
	t := ""
	if !r.Time.IsZero() && h.timefmt != "" {
		t = r.Time.Format(h.timefmt) + " "
	}

	g := ""
	if len(h.g) > 0 {
		g = strings.Join(h.g, "·") + ": "
	}

	color := ""
	width := h.getWidth()
	if width > 0 {
		color = zli.Colorize(" ", colors[r.Level])
	}

	pr := fmt.Sprintf("%s%s%-5s %s", color, t, r.Level, zli.Colorize(g+r.Message, zli.Bold))
	var (
		file string
		line int
	)
	if r.PC > 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		if frames != nil {
			f, _ := frames.Next()
			file, line = f.File, f.Line
		}
	}
	loc := fmt.Sprintf("%s:%d", strings.TrimPrefix(file, h.root), line)

	// Bit of a hack to shorten paths from dependencies:
	//
	//   02:22 INFO  msg   /home/martin/.cache/go/pkg/mod/github.com/riverqueue/river@v0.11.2/client.go:532
	//   02:24 INFO  msg   github.com/riverqueue/river@v0.11.2/client.go:532
	//
	// Not sure if there's really a good way to do this; we need to know the
	// GOMODCACHE value during build time, and I don't think that's available.
	//
	// TODO: only do this if not built with -trimpath.
	if i := strings.Index(loc, "pkg/mod/"); i > -1 {
		loc = loc[i+8:]
	}

	sep := "  "
	if width > 0 {
		l := width - termtext.Width(pr) - termtext.Width(loc)
		if l < 0 {
			l = 1
		}
		sep = strings.Repeat(" ", l)
	} else {
		loc = "[" + loc + "]"
	}
	fmt.Fprintln(h.w, pr+sep+loc)

	attr := make([]slog.Attr, 0, r.NumAttrs())
	w := 0
	r.Attrs(func(a slog.Attr) bool {
		if h.replAttr != nil {
			a = h.replAttr(h.g, a)
			if a.Equal(slog.Attr{}) {
				return true
			}
		}
		if h := termtext.Width(a.Key); h > w {
			w = h
		}
		attr = append(attr, a)
		return true
	})
	for _, a := range h.attr {
		if h.replAttr != nil {
			a = h.replAttr(h.g, a)
			if a.Equal(slog.Attr{}) {
				continue
			}
		}
		if h := termtext.Width(a.Key); h > w {
			w = h
		}
		attr = append(attr, a)
	}

	for _, a := range attr {
		if a.Key == "" {
			continue
		}

		var val string
		switch v := a.Value; v.Kind() {
		default:
			val = v.String()

		case slog.KindAny:
			m, ok := v.Any().(map[string]any)
			if !ok { // Not a map (e.g. slice, struct): just use string
				val = v.String()
			} else {
				var buf bytes.Buffer
				enc := json.NewEncoder(&buf)
				enc.SetEscapeHTML(false)
				if err := enc.Encode(m); err != nil {
					return err
				}
				f := jfmt.NewFormatter(100, "", "    ")
				var err error
				val, err = f.FormatString(buf.String())
				if err != nil {
					return err
				}
			}
		}

		fmt.Fprintf(h.w, "%s%s%s = %s\n", h.indent, a.Key,
			strings.Repeat(" ", w-termtext.Width(a.Key)),
			strings.ReplaceAll(strings.TrimRight(val, "\n"), "\n", "\n"+h.indent))
	}

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of both the
// receiver's attributes and the arguments.
//
// The Handler owns the slice: it may retain, modify or discard it.
func (h AlignedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attr = append(h.attr, attrs...)
	return h
}

// WithGroup returns a new Handler with the given group appended to the
// receiver's existing groups.
//
// The keys of all subsequent attributes, whether added by With or in a Record,
// should be qualified by the sequence of group names.
//
// How this qualification happens is up to the Handler, so long as this
// Handler's attribute keys differ from those of another Handler with a
// different sequence of group names.
//
// A Handler should treat WithGroup as starting a Group of Attrs that ends at
// the end of the log event. That is,
//
//	logger.WithGroup("s").LogAttrs(level, msg, slog.Int("a", 1), slog.Int("b", 2))
//
// should behave like
//
//	logger.LogAttrs(level, msg, slog.Group("s", slog.Int("a", 1), slog.Int("b", 2)))
func (h AlignedHandler) WithGroup(name string) slog.Handler {
	h.g = append(h.g, name)
	return h
}

// moduleRoot gets the full path to the module root directory.
//
// Returns empty string if it can't find a module.
//
// Copy from:
// https://github.com/arp242/zstd/blob/f20b0b1e56be7d3d6e019699dfa6425e50055010/zgo/zgo.go#L13
func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return ""
	}

	pdir := dir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)

		/// Parent directory is identical: we reached the top of the filesystem
		/// hierarchy and didn't find anything.
		if dir == pdir {
			return ""
		}
		pdir = dir
	}
}
