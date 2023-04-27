// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package diff computes differences between text files or strings.
package diff

import (
	"fmt"
	"sort"

	"github.com/pgavlin/text"
)

// Edit represents a change to a section of a document.
// The text within the specified span should be replaced by the supplied new text.
type Edit[T text.Text] struct {
	Start, End int // byte offsets of the region to replace
	New        T
}

func (e Edit[T]) String() string {
	return fmt.Sprintf("{Start:%d,End:%d,New:%s}", e.Start, e.End, string(e.New))
}

// Apply applies a sequence of edits to the src buffer and returns the
// result. Edits are applied in order of start offset; edits with the
// same start offset are applied in they order they were provided.
//
// Apply returns an error if any edit is out of bounds,
// or if any pair of edits is overlapping.
func Apply[T text.Text](src T, edits []Edit[T]) (T, error) {
	edits, size, err := validate(src, edits)
	if err != nil {
		return T(""), err
	}

	// Apply edits.
	out := make([]byte, 0, size)
	lastEnd := 0
	for _, edit := range edits {
		if lastEnd < edit.Start {
			out = append(out, src[lastEnd:edit.Start]...)
		}
		out = append(out, edit.New...)
		lastEnd = edit.End
	}
	out = append(out, src[lastEnd:]...)

	if len(out) != size {
		panic("wrong size")
	}

	return T(out), nil
}

// validate checks that edits are consistent with src,
// and returns the size of the patched output.
// It may return a different slice.
func validate[T text.Text](src T, edits []Edit[T]) ([]Edit[T], int, error) {
	if !sort.IsSorted(editsSort[T]{edits}) {
		edits = append([]Edit[T](nil), edits...)
		SortEdits(edits)
	}

	// Check validity of edits and compute final size.
	size := len(src)
	lastEnd := 0
	for _, edit := range edits {
		if !(0 <= edit.Start && edit.Start <= edit.End && edit.End <= len(src)) {
			return nil, 0, fmt.Errorf("diff has out-of-bounds edits")
		}
		if edit.Start < lastEnd {
			return nil, 0, fmt.Errorf("diff has overlapping edits")
		}
		size += len(edit.New) + edit.Start - edit.End
		lastEnd = edit.End
	}

	return edits, size, nil
}

// SortEdits orders a slice of Edits by (start, end) offset.
// This ordering puts insertions (end = start) before deletions
// (end > start) at the same point, but uses a stable sort to preserve
// the order of multiple insertions at the same point.
// (Apply detects multiple deletions at the same point as an error.)
func SortEdits[T text.Text](edits []Edit[T]) {
	sort.Stable(editsSort[T]{edits})
}

type editsSort[T text.Text] struct {
	edits []Edit[T]
}

func (a editsSort[T]) Len() int { return len(a.edits) }
func (a editsSort[T]) Less(i, j int) bool {
	if cmp := a.edits[i].Start - a.edits[j].Start; cmp != 0 {
		return cmp < 0
	}
	return a.edits[i].End < a.edits[j].End
}
func (a editsSort[T]) Swap(i, j int) { a.edits[i], a.edits[j] = a.edits[j], a.edits[i] }

// lineEdits expands and merges a sequence of edits so that each
// resulting edit replaces one or more complete lines.
// See ApplyEdits for preconditions.
func lineEdits[T text.Text, A text.Algorithms[T]](src T, edits []Edit[T]) ([]Edit[T], error) {
	var alg A

	edits, _, err := validate(src, edits)
	if err != nil {
		return nil, err
	}

	// Do all edits begin and end at the start of a line?
	// TODO(adonovan): opt: is this fast path necessary?
	// (Also, it complicates the result ownership.)
	for _, edit := range edits {
		if edit.Start >= len(src) || // insertion at EOF
			edit.Start > 0 && src[edit.Start-1] != '\n' || // not at line start
			edit.End > 0 && src[edit.End-1] != '\n' { // not at line start
			goto expand
		}
	}
	return edits, nil // aligned

expand:
	expanded := make([]Edit[T], 0, len(edits)) // a guess
	prev := edits[0]
	// TODO(adonovan): opt: start from the first misaligned edit.
	// TODO(adonovan): opt: avoid quadratic cost of string += string.
	for _, edit := range edits[1:] {
		between := src[prev.End:edit.Start]
		if !alg.ContainsAny(between, "\n") {
			// overlapping lines: combine with previous edit.
			prev.New = alg.Join([]T{prev.New, between, edit.New}, T(""))
			prev.End = edit.End
		} else {
			// non-overlapping lines: flush previous edit.
			expanded = append(expanded, expandEdit[T, A](prev, src))
			prev = edit
		}
	}
	return append(expanded, expandEdit[T, A](prev, src)), nil // flush final edit
}

// expandEdit returns edit expanded to complete whole lines.
func expandEdit[T text.Text, A text.Algorithms[T]](edit Edit[T], src T) Edit[T] {
	var alg A

	// Expand start left to start of line.
	// (delta is the zero-based column number of of start.)
	start := edit.Start
	if delta := start - 1 - alg.LastIndexByte(src[:start], '\n'); delta > 0 {
		edit.Start -= delta
		edit.New = alg.Concat(src[start-delta:start], edit.New)
	}

	// Expand end right to end of line.
	end := edit.End
	if nl := alg.IndexByte(src[end:], '\n'); nl < 0 {
		edit.End = len(src) // extend to EOF
	} else {
		edit.End = end + nl + 1 // extend beyond \n
	}
	edit.New = alg.Concat(edit.New, src[end:edit.End])

	return edit
}
