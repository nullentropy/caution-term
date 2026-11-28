package term

import (
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	caution "github.com/nullentropy/caution"
)

type screen struct {
	t     *testing.T
	c     *websocket.Conn
	grid  int
	seq   int
	lines map[int][]map[string]any
}

func (sc *screen) send(ev string, value any) {
	sc.t.Helper()
	if err := sc.c.WriteJSON(map[string]any{"t": "ev", "id": sc.grid, "ev": ev, "value": value, "seq": sc.seq}); err != nil {
		sc.t.Fatal(err)
	}
}

func (sc *screen) row(y int) string {
	var b []rune
	for _, r := range sc.lines[y] {
		x := int(r["x"].(float64))
		for len(b) < x {
			b = append(b, ' ')
		}
		b = append(b[:x], []rune(r["t"].(string))...)
	}
	return strings.TrimRight(string(b), " ")
}

func (sc *screen) bgAt(y, x int) string {
	for _, r := range sc.lines[y] {
		rx := int(r["x"].(float64))
		if x >= rx && x < rx+len([]rune(r["t"].(string))) {
			bg, _ := r["b"].(string)
			return bg
		}
	}
	return ""
}

func (sc *screen) apply(p map[string]any) {
	for k, v := range p {
		var y int
		if n, err := parseLine(k); err == nil {
			y = n
		} else {
			continue
		}
		var runs []map[string]any
		for _, r := range v.([]any) {
			runs = append(runs, r.(map[string]any))
		}
		sc.lines[y] = runs
	}
}

func parseLine(k string) (int, error) {
	row, ok := strings.CutPrefix(k, "line:")
	if !ok {
		return 0, strconv.ErrSyntax
	}
	return strconv.Atoi(row)
}

func (sc *screen) until(desc string, ok func() bool) {
	sc.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			sc.t.Fatalf("timed out waiting for %s; screen:\n%s", desc, sc.dump())
		}
		sc.c.SetReadDeadline(time.Now().Add(time.Second))
		var m map[string]any
		if err := sc.c.ReadJSON(&m); err != nil {
			continue
		}
		if s, ok := m["seq"].(float64); ok {
			sc.seq = int(s)
		}
		ops, _ := m["ops"].([]any)
		for _, o := range ops {
			op := o.(map[string]any)
			if op["op"] == "set" && int(op["id"].(float64)) == sc.grid {
				sc.apply(op["p"].(map[string]any))
			}
		}
	}
}

func (sc *screen) dump() string {
	var b strings.Builder
	for y := 0; y < 10; y++ {
		b.WriteString(sc.row(y) + "\n")
	}
	return b.String()
}

func findGrid(n map[string]any) (int, map[string]any) {
	if n["type"] == "termgrid" {
		return int(n["id"].(float64)), n["p"].(map[string]any)
	}
	kids, _ := n["kids"].([]any)
	for _, k := range kids {
		if id, p := findGrid(k.(map[string]any)); id != 0 {
			return id, p
		}
	}
	return 0, nil
}

// caution registers its handler once per process and keeps the first mount
// so every test shares one server and hands it options
var (
	serveOnce sync.Once
	serveAddr string
	serveErr  error
	nextOpts  = make(chan Options, 1)
	nextTerm  = make(chan *Term, 1)
)

func open(t *testing.T, o Options) (*screen, *Term) {
	t.Helper()
	serveOnce.Do(func() {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			serveErr = err
			return
		}
		serveAddr = l.Addr().String()
		go caution.ServeListener(l, func(s *caution.Session) *caution.Node {
			w := NewWindow(s, WindowOptions{})
			nextTerm <- w.Open(<-nextOpts)
			return w.Node()
		}, caution.Options{})
	})
	if serveErr != nil {
		t.Fatal(serveErr)
	}
	nextOpts <- o

	c, _, err := websocket.DefaultDialer.Dial("ws://"+serveAddr+"/ws?vw=800&vh=600", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	var m map[string]any
	if err := c.ReadJSON(&m); err != nil {
		t.Fatal(err)
	}
	id, _ := findGrid(m["root"].(map[string]any))
	if id == 0 {
		t.Fatalf("no termgrid in mount: %v", m)
	}
	sc := &screen{t: t, c: c, grid: id, lines: map[int][]map[string]any{}}
	if s, ok := m["seq"].(float64); ok {
		sc.seq = int(s)
	}
	return sc, <-nextTerm
}

// child of TestOneWriteRendersAsOneFrame, one write much bigger than a pty
// read with row 0 going A then B
func init() {
	if os.Getenv("CAUTION_TERM_ONE_WRITE") == "" {
		return
	}
	var b strings.Builder
	b.WriteString("\x1b[2J\x1b[1;1HA")
	for b.Len() < 32<<10 {
		b.WriteString("\x1b[6;1Hfiller")
	}
	b.WriteString("\x1b[1;1HB")
	time.Sleep(500 * time.Millisecond)
	os.Stdout.WriteString(b.String())
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func TestOneWriteRendersAsOneFrame(t *testing.T) {
	sc, _ := open(t, Options{Argv: []string{os.Args[0]}, Env: []string{"CAUTION_TERM_ONE_WRITE=1"}})
	sc.send("grid-size", map[string]any{"cols": 20, "rows": 10})
	sawA := false
	sc.until("the end of the write", func() bool {
		sawA = sawA || sc.row(0) == "A"
		return sc.row(0) == "B"
	})
	if sawA {
		t.Fatal("drew row 0 as A, the middle of a single write")
	}
}

func TestScrollbackAndSelectionWorkOnRealOutput(t *testing.T) {
	sc, term := open(t, Options{Argv: []string{"sh", "-c", `seq 1 60; printf 'alpha beta\n'; sleep 30`}})
	sc.send("grid-size", map[string]any{"cols": 20, "rows": 10})
	sc.until("the program's output", func() bool { return sc.row(8) == "alpha beta" })
	if got := sc.row(0); got != "53" {
		t.Fatalf("top row = %q, want 53", got)
	}

	sc.send("term-mouse", map[string]any{"kind": "wheel", "x": 0, "y": 0, "dy": -5})
	sc.until("scrolling back five lines", func() bool { return sc.row(0) == "48" })

	sc.send("term-mouse", map[string]any{"kind": "wheel", "x": 0, "y": 0, "dy": 5})
	sc.until("scrolling back to the end", func() bool { return sc.row(0) == "53" })

	selected := func() string {
		got := make(chan string, 1)
		term.s.Update(func() { got <- term.SelectedText() })
		return <-got
	}

	sc.send("term-mouse", map[string]any{"kind": "press", "x": 7, "y": 8, "button": 1, "count": 2})
	sc.send("term-mouse", map[string]any{"kind": "release", "x": 7, "y": 8, "button": 1})
	sc.until("the word highlighted", func() bool { return sc.bgAt(8, 6) == selectionBg })
	if got := selected(); got != "beta" {
		t.Fatalf("double-click selected %q, want beta", got)
	}
	if bg := sc.bgAt(8, 0); bg == selectionBg {
		t.Fatalf("alpha is highlighted after selecting beta")
	}

	sc.send("term-mouse", map[string]any{"kind": "press", "x": 0, "y": 7, "button": 1, "count": 1})
	sc.send("term-mouse", map[string]any{"kind": "drag", "x": 4, "y": 8, "button": 1})
	sc.send("term-mouse", map[string]any{"kind": "release", "x": 4, "y": 8, "button": 1})
	sc.until("the drag highlighted", func() bool { return sc.bgAt(7, 0) == selectionBg && sc.bgAt(8, 6) != selectionBg })
	if got := selected(); got != "60\nalpha" {
		t.Fatalf("drag selected %q, want \"60\\nalpha\"", got)
	}
}
