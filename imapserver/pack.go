package imapserver

import (
	"fmt"
	"io"

	"github.com/mjl-/mox/mlog"
)

type token interface {
	pack(c *conn) string
	writeTo(c *conn, w io.Writer)
}

type bare string

func (t bare) pack(c *conn) string {
	return string(t)
}

func (t bare) writeTo(c *conn, w io.Writer) {
	w.Write([]byte(t.pack(c)))
}

type niltoken struct{}

var nilt niltoken

func (t niltoken) pack(c *conn) string {
	return "NIL"
}

func (t niltoken) writeTo(c *conn, w io.Writer) {
	w.Write([]byte(t.pack(c)))
}

func nilOrString(s string) token {
	if s == "" {
		return nilt
	}
	return string0(s)
}

type string0 string

// ../rfc/9051:7081
// ../rfc/9051:6856 ../rfc/6855:153
func (t string0) pack(c *conn) string {
	r := `"`
	for _, ch := range t {
		if ch == '\x00' || ch == '\r' || ch == '\n' || ch > 0x7f && !c.utf8strings() {
			return syncliteral(t).pack(c)
		}
		if ch == '\\' || ch == '"' {
			r += `\`
		}
		r += string(ch)
	}
	r += `"`
	return r
}

func (t string0) writeTo(c *conn, w io.Writer) {
	w.Write([]byte(t.pack(c)))
}

type dquote string

func (t dquote) pack(c *conn) string {
	r := `"`
	for _, c := range t {
		if c == '\\' || c == '"' {
			r += `\`
		}
		r += string(c)
	}
	r += `"`
	return r
}

func (t dquote) writeTo(c *conn, w io.Writer) {
	w.Write([]byte(t.pack(c)))
}

type syncliteral string

func (t syncliteral) pack(c *conn) string {
	return fmt.Sprintf("{%d}\r\n", len(t)) + string(t)
}

func (t syncliteral) writeTo(c *conn, w io.Writer) {
	fmt.Fprintf(w, "{%d}\r\n", len(t))
	w.Write([]byte(t))
}

// data from reader with known size.
type readerSizeSyncliteral struct {
	r    io.Reader
	size int64
}

func (t readerSizeSyncliteral) pack(c *conn) string {
	buf, err := io.ReadAll(t.r)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("{%d}\r\n", t.size) + string(buf)
}

func (t readerSizeSyncliteral) writeTo(c *conn, w io.Writer) {
	fmt.Fprintf(w, "{%d}\r\n", t.size)
	defer c.xtrace(mlog.LevelTracedata)()
	if _, err := io.Copy(w, io.LimitReader(t.r, t.size)); err != nil {
		panic(err)
	}
}

// data from reader without known size.
type readerSyncliteral struct {
	r io.Reader
}

func (t readerSyncliteral) pack(c *conn) string {
	buf, err := io.ReadAll(t.r)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("{%d}\r\n", len(buf)) + string(buf)
}

func (t readerSyncliteral) writeTo(c *conn, w io.Writer) {
	buf, err := io.ReadAll(t.r)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "{%d}\r\n", len(buf))
	defer c.xtrace(mlog.LevelTracedata)()
	_, err = w.Write(buf)
	if err != nil {
		panic(err)
	}
}

// list with tokens space-separated
type listspace []token

func (t listspace) pack(c *conn) string {
	s := "("
	for i, e := range t {
		if i > 0 {
			s += " "
		}
		s += e.pack(c)
	}
	s += ")"
	return s
}

func (t listspace) writeTo(c *conn, w io.Writer) {
	fmt.Fprint(w, "(")
	for i, e := range t {
		if i > 0 {
			fmt.Fprint(w, " ")
		}
		e.writeTo(c, w)
	}
	fmt.Fprint(w, ")")
}

// Concatenated tokens, no spaces or list syntax.
type concat []token

func (t concat) pack(c *conn) string {
	var s string
	for _, e := range t {
		s += e.pack(c)
	}
	return s
}

func (t concat) writeTo(c *conn, w io.Writer) {
	for _, e := range t {
		e.writeTo(c, w)
	}
}

type astring string

func (t astring) pack(c *conn) string {
	if len(t) == 0 {
		return string0(t).pack(c)
	}
next:
	for _, ch := range t {
		for _, x := range atomChar {
			if ch == x {
				continue next
			}
		}
		return string0(t).pack(c)
	}
	return string(t)
}

func (t astring) writeTo(c *conn, w io.Writer) {
	w.Write([]byte(t.pack(c)))
}

type number uint32

func (t number) pack(c *conn) string {
	return fmt.Sprintf("%d", t)
}

func (t number) writeTo(c *conn, w io.Writer) {
	w.Write([]byte(t.pack(c)))
}
