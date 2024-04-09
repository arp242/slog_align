package slog_align

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"zgo.at/zli"
)

func run(w io.Writer) {
	d := time.Date(2023, 03, 20, 8, 26, 0, 0, time.UTC)
	testTime = &d

	h := NewAlignedHandler(w)
	h.width = 80
	zli.WantColor = true
	slog.SetDefault(slog.New(h))

	slog.Error("error")
	slog.Warn("warn")
	slog.Info("info")
	slog.Debug("debug")

	l := slog.
		With("str", "foo").
		With("int", 123).
		With("map", map[string]any{
			"key":     "value",
			"another": true,
			"struct":  struct{ s []int }{[]int{1, 2, 3}},
		}).
		With("slice", []string{"a", "b"}).
		With("struct", struct {
			s string
			i int
		}{"asd", 123})
	l.Info("info")

	l = l.WithGroup("group")
	l.Info("info")
}

// d := time.Date(2023, 03, 20, 8, 26, 0, 0, time.UTC)
var want = `
[48;5;210m [0m08:26 ERROR [1merror[0m                                         slog_align_test.go:23
[48;5;214m [0m08:26 WARN  [1mwarn[0m                                          slog_align_test.go:24
[48;5;51m [0m08:26 INFO  [1minfo[0m                                          slog_align_test.go:25
[48;5;250m [0m08:26 DEBUG [1mdebug[0m                                         slog_align_test.go:26
[48;5;51m [0m08:26 INFO  [1minfo[0m                                          slog_align_test.go:41
             str    = foo
             int    = 123
             map    = map[another:true key:value struct:{[1 2 3]}]
             slice  = [a b]
             struct = {asd 123}
[48;5;51m [0m08:26 INFO  group: [1minfo[0m                                   slog_align_test.go:44
             str    = foo
             int    = 123
             map    = map[another:true key:value struct:{[1 2 3]}]
             slice  = [a b]
             struct = {asd 123}
`[1:]

func TestAlignedHandler(t *testing.T) {
	var b bytes.Buffer
	run(&b)
	if b.String() != want {
		//fmt.Println(ztest.Diff(b.String(), want))
		t.Fatalf("\nhave:\n%q\n\nwant:\n%q\n", b.String(), want)
	}

	run(os.Stdout)
}
