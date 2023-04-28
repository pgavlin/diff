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
func Text[T text.String](before, after T) []Edit[T] {
	return diffText(before, after, false)
}

// Binary computes the differences between two texts. The texts are treated as
// binary data. The resulting edits do not respect rune boundaries.
func Binary[T text.String](before, after T) []Edit[T] {
	return diffText(before, after, true)
}

func diffText[T text.String](before, after T, binary bool) []Edit[T] {
	if text.Compare(before, after) == 0 {
		return nil // common case
	}

	if binary || isASCII(before) && isASCII(after) {
		return diffASCII(before, after)
	}
	return diffRunes[T](text.ToRunes(before), text.ToRunes(after))
}

func diffASCII[T text.String](before, after T) []Edit[T] {
	diffs := lcs.DiffText(before, after)

	// Convert from LCS diffs.
	res := make([]Edit[T], len(diffs))
	for i, d := range diffs {
		res[i] = Edit[T]{d.Start, d.End, after[d.ReplStart:d.ReplEnd]}
	}
	return res
}

func diffRunes[T text.String](before, after []rune) []Edit[T] {
	diffs := lcs.DiffRunes(before, after)

	// The diffs returned by the lcs package use indexes
	// into whatever slice was passed in.
	// Convert rune offsets to byte offsets.
	res := make([]Edit[T], len(diffs))
	lastEnd := 0
	utf8Len := 0
	for i, d := range diffs {
		utf8Len += runesLen(before[lastEnd:d.Start]) // text between edits
		start := utf8Len
		utf8Len += runesLen(before[d.Start:d.End]) // text deleted by this edit
		res[i] = Edit[T]{start, utf8Len, text.ToString[T](after[d.ReplStart:d.ReplEnd])}
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

func isASCII[T text.String](s T) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
