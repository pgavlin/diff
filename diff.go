// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package gotextdiff supports a pluggable diff algorithm.
package gotextdiff

import (
	"sort"

	"github.com/pgavlin/gotextdiff/span"
	"github.com/pgavlin/gotextdiff/text"
)

type Text = span.Text

type Rope[T Text] struct {
	text []T
}

func NewRope[T Text](text ...T) Rope[T] {
	return Rope[T]{text: text}
}

func (r Rope[T]) append(t ...T) Rope[T] {
	return Rope[T]{text: append(r.text, t...)}
}

func (r Rope[T]) String() string {
	return string(r.Text())
}

func (r Rope[T]) Text() T {
	if text.UseStrings[T]() {
		var b text.StringBuilder[T]
		appendRope(&b, r)
		return b.Text()
	}

	var b text.BytesBuilder[T]
	appendRope(&b, r)
	return b.Text()
}

func appendRope[T Text, B text.Builder[T]](b B, r Rope[T]) {
	for _, t := range r.text {
		b.WriteText(t)
	}
}

// TextEdit represents a change to a section of a document.
// The text within the specified span should be replaced by the supplied new text.
type TextEdit[T Text] struct {
	Span    span.Span
	NewText Rope[T]
}

// ComputeEdits is the type for a function that produces a set of edits that
// convert from the before content to the after content.
type ComputeEdits[T Text] func(uri span.URI, before, after T) []TextEdit[T]

// SortTextEdits attempts to order all edits by their starting points.
// The sort is stable so that edits with the same starting point will not
// be reordered.
func SortTextEdits[T Text](d []TextEdit[T]) {
	// Use a stable sort to maintain the order of edits inserted at the same position.
	sort.SliceStable(d, func(i int, j int) bool {
		return span.Compare(d[i].Span, d[j].Span) < 0
	})
}

// ApplyEdits applies the set of edits to the before and returns the resulting
// content.
// It may panic or produce garbage if the edits are not valid for the provided
// before content.
func ApplyEdits[T Text](before T, edits []TextEdit[T]) T {
	if text.UseStrings[T]() {
		var builder text.StringBuilder[T]
		return T(applyEdits(&builder, before, edits))
	}

	var builder text.BytesBuilder[T]
	return T(applyEdits(&builder, before, edits))
}

func applyEdits[T Text, B text.Builder[T]](after B, before T, edits []TextEdit[T]) T {
	// Preconditions:
	//   - all of the edits apply to before
	//   - and all the spans for each TextEdit have the same URI
	if len(edits) == 0 {
		return before
	}
	edits, _ = prepareEdits(before, edits)
	last := 0
	for _, edit := range edits {
		start := edit.Span.Start().Offset()
		if start > last {
			after.WriteText(before[last:start])
			last = start
		}
		appendRope(after, edit.NewText)
		last = edit.Span.End().Offset()
	}
	if last < len(before) {
		after.WriteText((before[last:]))
	}
	return after.Text()
}

// LineEdits takes a set of edits and expands and merges them as necessary
// to ensure that there are only full line edits left when it is done.
func LineEdits[T Text](before T, edits []TextEdit[T]) []TextEdit[T] {
	if text.UseStrings[T]() {
		return doLineEdits[T, text.Strings[T]](before, edits)
	}

	return doLineEdits[T, text.Bytes[T]](before, edits)
}

func doLineEdits[T Text, A text.Algorithms[T]](before T, edits []TextEdit[T]) []TextEdit[T] {
	if len(edits) == 0 {
		return nil
	}
	edits, partial := prepareEdits(before, edits)
	if partial {
		edits = lineEdits[T, A](before, edits)
	}
	return edits
}

// prepareEdits returns a sorted copy of the edits
func prepareEdits[T Text](before T, edits []TextEdit[T]) ([]TextEdit[T], bool) {
	partial := false
	c := span.NewContentConverter("", before)
	copied := make([]TextEdit[T], len(edits))
	for i, edit := range edits {
		edit.Span, _ = edit.Span.WithAll(c)
		copied[i] = edit
		partial = partial ||
			edit.Span.Start().Offset() >= len(before) ||
			edit.Span.Start().Column() > 1 || edit.Span.End().Column() > 1
	}
	SortTextEdits(copied)
	return copied, partial
}

// lineEdits rewrites the edits to always be full line edits
func lineEdits[T Text, A text.Algorithms[T]](before T, edits []TextEdit[T]) []TextEdit[T] {
	adjusted := make([]TextEdit[T], 0, len(edits))
	current := TextEdit[T]{Span: span.Invalid}
	for _, edit := range edits {
		if current.Span.IsValid() && edit.Span.Start().Line() <= current.Span.End().Line() {
			// overlaps with the current edit, need to combine
			// first get the gap from the previous edit
			gap := before[current.Span.End().Offset():edit.Span.Start().Offset()]
			// now add the text of this edit
			current.NewText = current.NewText.append(gap).append(edit.NewText.text...)
			// and then adjust the end position
			current.Span = span.New(current.Span.URI(), current.Span.Start(), edit.Span.End())
		} else {
			// does not overlap, add previous run (if there is one)
			adjusted = addEdit[T, A](before, adjusted, current)
			// and then remember this edit as the start of the next run
			current = edit
		}
	}
	// add the current pending run if there is one
	return addEdit[T, A](before, adjusted, current)
}

func addEdit[T Text, A text.Algorithms[T]](before T, edits []TextEdit[T], edit TextEdit[T]) []TextEdit[T] {
	var alg A

	if !edit.Span.IsValid() {
		return edits
	}
	// if edit is partial, expand it to full line now
	start := edit.Span.Start()
	end := edit.Span.End()
	if start.Column() > 1 {
		// prepend the text and adjust to start of line
		delta := start.Column() - 1
		start = span.NewPoint(start.Line(), 1, start.Offset()-delta)
		edit.Span = span.New(edit.Span.URI(), start, end)
		edit.NewText = NewRope(before[start.Offset() : start.Offset()+delta]).append(edit.NewText.text...)
	}
	if start.Offset() >= len(before) && start.Line() > 1 && before[len(before)-1] != '\n' {
		// after end of file that does not end in eol, so join to last line of file
		// to do this we need to know where the start of the last line was
		eol := alg.LastIndexByte(before, '\n')
		if eol < 0 {
			// file is one non terminated line
			eol = 0
		}
		delta := len(before) - eol
		start = span.NewPoint(start.Line()-1, 1, start.Offset()-delta)
		edit.Span = span.New(edit.Span.URI(), start, end)
		edit.NewText = NewRope(before[start.Offset() : start.Offset()+delta]).append(edit.NewText.text...)
	}
	if end.Column() > 1 {
		remains := before[end.Offset():]
		eol := alg.IndexByte(remains, '\n')
		if eol < 0 {
			eol = len(remains)
		} else {
			eol++
		}
		end = span.NewPoint(end.Line()+1, 1, end.Offset()+eol)
		edit.Span = span.New(edit.Span.URI(), start, end)
		edit.NewText = edit.NewText.append(remains[:eol])
	}
	edits = append(edits, edit)
	return edits
}
