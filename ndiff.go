// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff

import (
	"unicode/utf8"

	"github.com/pgavlin/diff/lcs"

	"github.com/pgavlin/text"
)

// Text computes the differences between two texts.
// The resulting edits respect rune boundaries.
func Text[S1, S2 text.String](before S1, after S2) []Edit[S2] {
	return diffText(before, after, false)
}

// Lines computes the line differences between two texts.
func Lines[S1, S2 text.String](before S1, after S2) []Edit[S2] {
	beforeLines, afterLines := splitLines(before), splitLines(after)

	diffs := lcs.DiffLines(beforeLines, afterLines)

	// Build tables mapping line number to offset.
	beforeLineOffsets, afterLineOffsets := lineOffsets(beforeLines), lineOffsets(afterLines)

	edits := make([]Edit[S2], 0, len(diffs))
	for _, diff := range diffs {
		start, end := beforeLineOffsets[diff.Start], beforeLineOffsets[diff.End]
		replStart, replEnd := afterLineOffsets[diff.ReplStart], afterLineOffsets[diff.ReplEnd]
		edits = append(edits, Edit[S2]{Start: start, End: end, New: after[replStart:replEnd]})
	}
	return edits
}

// Binary computes the differences between two texts. The texts are treated as
// binary data. The resulting edits do not respect rune boundaries.
func Binary[S1, S2 text.String](before S1, after S2) []Edit[S2] {
	return diffText(before, after, true)
}

func diffText[S1, S2 text.String](before S1, after S2, binary bool) []Edit[S2] {
	if text.Equal(before, after) {
		return nil // common case
	}

	if binary || isASCII(before) && isASCII(after) {
		return diffASCII(before, after)
	}
	return diffRunes[S2](text.ToRunes(before), text.ToRunes(after))
}

func diffASCII[S1, S2 text.String](before S1, after S2) []Edit[S2] {
	diffs := lcs.DiffText(before, after)

	// Convert from LCS diffs.
	res := make([]Edit[S2], len(diffs))
	for i, d := range diffs {
		res[i] = Edit[S2]{d.Start, d.End, after[d.ReplStart:d.ReplEnd]}
	}
	return res
}

func diffRunes[S text.String](before, after []rune) []Edit[S] {
	diffs := lcs.DiffRunes(before, after)

	// The diffs returned by the lcs package use indexes
	// into whatever slice was passed in.
	// Convert rune offsets to byte offsets.
	res := make([]Edit[S], len(diffs))
	lastEnd := 0
	utf8Len := 0
	for i, d := range diffs {
		utf8Len += runesLen(before[lastEnd:d.Start]) // text between edits
		start := utf8Len
		utf8Len += runesLen(before[d.Start:d.End]) // text deleted by this edit
		res[i] = Edit[S]{start, utf8Len, text.ToString[S](after[d.ReplStart:d.ReplEnd])}
		lastEnd = d.End
	}
	return res
}

// runesLen returns the length in bytes of the UTF-8 encoding of runes.
func runesLen(runes []rune) (len int) {
	for _, r := range runes {
		len += utf8.RuneLen(r)
	}
	return len
}

func isASCII[S text.String](s S) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
