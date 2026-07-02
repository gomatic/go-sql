// Package doc is a small Wadler/Prettier-style document algebra: you build an
// immutable layout tree out of [Text], [Concat], [Group], [Indent], and the
// line primitives, then [Render] it at a target width. A [Group] lays its
// contents out on one line when they fit and breaks them across lines when they
// don't, so callers describe structure and let the renderer own wrapping.
//
// Documents are immutable values: every constructor returns a fresh [Doc] and
// none of them retain or mutate their arguments, so a Doc is safe to share and
// reuse.
package doc

import "strings"

// indentStep is how many spaces one [Indent] level adds.
const indentStep = 2

// kind tags the shape of a [Doc] node.
type kind int

const (
	kindText     kind = iota // a literal string
	kindConcat               // a sequence of child docs
	kindLine                 // a space when flat, a newline when broken
	kindSoftline             // empty when flat, a newline when broken
	kindHardline             // always a newline; forces the enclosing group to break
	kindIndent               // its child, indented one step deeper
	kindGroup                // its child, flat if it fits else broken
)

// Doc is an immutable layout document. The zero value renders as empty text.
// Fields are ordered so the garbage collector's pointer-scan prefix is as short
// as possible.
type Doc struct {
	text     string
	children []Doc
	kind     kind
}

// Text is a document that renders literally as s, on a single line.
func Text(s string) Doc {
	return Doc{kind: kindText, text: s}
}

// Concat is the documents rendered one after another with nothing between them.
func Concat(docs ...Doc) Doc {
	return Doc{kind: kindConcat, children: docs}
}

// Group renders d flat (every [Line] a space, every [Softline] empty) when it
// fits the remaining width, and broken (every line primitive a newline) when it
// doesn't. A [Hardline] anywhere inside forces the broken layout.
func Group(d Doc) Doc {
	return Doc{kind: kindGroup, children: []Doc{d}}
}

// Indent renders d one indentation step deeper, which only shows up on the lines
// that follow a break inside d.
func Indent(d Doc) Doc {
	return Doc{kind: kindIndent, children: []Doc{d}}
}

// Line renders as a single space when its group is flat and as a newline when
// the group is broken.
func Line() Doc { return Doc{kind: kindLine} }

// Softline renders as nothing when its group is flat and as a newline when the
// group is broken.
func Softline() Doc { return Doc{kind: kindSoftline} }

// Hardline always renders as a newline and forces every enclosing [Group] to
// break.
func Hardline() Doc { return Doc{kind: kindHardline} }

// mode is whether a group is being laid out flat or broken.
type mode int

const (
	modeFlat mode = iota
	modeBreak
)

// frame is one pending document to emit, carrying the indentation it inherited
// and the mode of the group it belongs to.
type frame struct {
	doc    Doc
	indent int
	mode   mode
}

// Render lays d out as text, breaking groups that would otherwise exceed width
// columns.
func Render(d Doc, width int) string {
	var out strings.Builder
	col := 0
	stack := []frame{{doc: d, indent: 0, mode: modeBreak}}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		col, stack = step(&out, top, column(col), stack, lineWidth(width))
	}
	return out.String()
}

// column is the output column the renderer is currently at.
type column int

// lineWidth is the column budget a rendered line must fit within.
type lineWidth int

// step emits one frame, pushing any children back onto the stack, and reports
// the new column and stack.
func step(out *strings.Builder, f frame, col column, stack []frame, width lineWidth) (int, []frame) {
	switch f.kind() {
	case kindText:
		_, _ = out.WriteString(f.doc.text)
		return int(col) + len(f.doc.text), stack
	case kindConcat, kindIndent:
		return int(col), pushChildren(f, stack)
	case kindGroup:
		return int(col), pushGroup(f, stack, int(width), int(col))
	default:
		return emitLine(out, f, col), stack
	}
}

// kind reports the node kind of the framed document.
func (f frame) kind() kind { return f.doc.kind }

// pushChildren puts a concat's or indent's children back on the stack in
// reverse so they emit left to right, deepening the indent for an indent node.
func pushChildren(f frame, stack []frame) []frame {
	indent := f.indent
	if f.doc.kind == kindIndent {
		indent += indentStep
	}
	for i := len(f.doc.children) - 1; i >= 0; i-- {
		stack = append(stack, frame{doc: f.doc.children[i], indent: indent, mode: f.mode})
	}
	return stack
}

// pushGroup chooses flat or broken for the group's child and pushes it. A child
// that carries a hardline always breaks; otherwise it stays flat when it fits.
func pushGroup(f frame, stack []frame, width, col int) []frame {
	child := f.doc.children[0]
	groupMode := modeBreak
	flat := frame{doc: child, indent: f.indent, mode: modeFlat}
	if !hasHardline(child) && fits(width-col, flat, stack) {
		groupMode = modeFlat
	}
	return append(stack, frame{doc: child, indent: f.indent, mode: groupMode})
}

// hasHardline reports whether d contains a hardline anywhere, which forces every
// enclosing group to break.
func hasHardline(d Doc) bool {
	if d.kind == kindHardline {
		return true
	}
	for _, c := range d.children {
		if hasHardline(c) {
			return true
		}
	}
	return false
}

// fits reports whether the flat layout of f, followed by the continuation in
// rest, stays within remaining columns up to the first newline. A line in break
// mode or a hardline is that newline: everything up to it fits.
func fits(remaining int, f frame, rest []frame) bool {
	stack := append(append([]frame{}, rest...), f)
	for remaining >= 0 && len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		var newline bool
		remaining, stack, newline = fitsStep(top, remainingWidth(remaining), stack)
		if newline {
			return true
		}
	}
	return remaining >= 0
}

// remainingWidth is how many columns are left on the current line.
type remainingWidth int

// fitsStep consumes one frame for [fits], returning the remaining width, the
// updated work stack, and whether this frame ended the current line.
func fitsStep(f frame, remaining remainingWidth, stack []frame) (int, []frame, bool) {
	switch f.doc.kind {
	case kindText:
		return int(remaining) - len(f.doc.text), stack, false
	case kindConcat, kindIndent:
		return int(remaining), pushChildren(f, stack), false
	case kindGroup:
		return int(remaining), append(stack, frame{doc: f.doc.children[0], indent: f.indent, mode: f.mode}), false
	case kindLine:
		return fitsLine(f, remaining, stack)
	case kindSoftline:
		return int(remaining), stack, f.mode == modeBreak
	default:
		return int(remaining), stack, true
	}
}

// fitsLine handles a [Line] inside [fits]: a break-mode line is a newline; a
// flat one costs the single space it would render.
func fitsLine(f frame, remaining remainingWidth, stack []frame) (int, []frame, bool) {
	if f.mode == modeBreak {
		return int(remaining), stack, true
	}
	return int(remaining) - 1, stack, false
}

// emitLine renders a line primitive: a space or empty when flat, a newline plus
// indentation when broken, and reports the resulting column.
func emitLine(out *strings.Builder, f frame, col column) int {
	if f.mode == modeFlat && f.doc.kind != kindHardline {
		if f.doc.kind == kindLine {
			_, _ = out.WriteString(" ")
			return int(col) + 1
		}
		return int(col)
	}
	_, _ = out.WriteString("\n")
	_, _ = out.WriteString(strings.Repeat(" ", f.indent))
	return f.indent
}
