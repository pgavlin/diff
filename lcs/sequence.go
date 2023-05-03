// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lcs

import (
	"unsafe"

	"github.com/pgavlin/text"
)

// This file defines the abstract sequence over which the LCS algorithm operates.

func asString[S text.String](s S) string {
	return *(*string)(unsafe.Pointer(&s))
}

// sequences abstracts a pair of sequences, A and B.
type sequences interface {
	lengths() (int, int)                    // len(A), len(B)
	commonPrefixLen(ai, aj, bi, bj int) int // len(commonPrefix(A[ai:aj], B[bi:bj]))
	commonSuffixLen(ai, aj, bi, bj int) int // len(commonSuffix(A[ai:aj], B[bi:bj]))
}

func textSeqs[S1, S2 text.String](a S1, b S2) sequences {
	return stringSeqs{a: asString(a), b: asString(b)}
}

type sliceSeqs[T comparable, S1 ~[]T, S2 ~[]T] struct {
	a S1
	b S2
}

func (s sliceSeqs[T, S1, S2]) lengths() (int, int) { return len(s.a), len(s.b) }
func (s sliceSeqs[T, S1, S2]) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenSlices(s.a[ai:aj], s.b[bi:bj])
}
func (s sliceSeqs[T, S1, S2]) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenSlices(s.a[ai:aj], s.b[bi:bj])
}

type anySliceSeqs[T1, T2 any, S1 ~[]T1, S2 ~[]T2, C EqualsComparer[T1, T2]] struct {
	a S1
	b S2
	c C
}

func (s anySliceSeqs[T1, T2, S1, S2, C]) lengths() (int, int) { return len(s.a), len(s.b) }
func (s anySliceSeqs[T1, T2, S1, S2, C]) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenAnySlices(s.a[ai:aj], s.b[bi:bj], s.c)
}
func (s anySliceSeqs[T1, T2, S1, S2, C]) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenAnySlices(s.a[ai:aj], s.b[bi:bj], s.c)
}

type lineSeqs[S1, S2 text.String] struct {
	a []S1
	b []S2
}

func (s lineSeqs[S1, S2]) lengths() (int, int) { return len(s.a), len(s.b) }
func (s lineSeqs[S1, S2]) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenLines(s.a[ai:aj], s.b[bi:bj])
}
func (s lineSeqs[S1, S2]) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenLines(s.a[ai:aj], s.b[bi:bj])
}

type stringSeqs struct{ a, b string }

func (s stringSeqs) lengths() (int, int) { return len(s.a), len(s.b) }
func (s stringSeqs) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenString(s.a[ai:aj], s.b[bi:bj])
}
func (s stringSeqs) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenString(s.a[ai:aj], s.b[bi:bj])
}

// The explicit capacity in s[i:j:j] leads to more efficient code.

type bytesSeqs struct{ a, b []byte }

func (s bytesSeqs) lengths() (int, int) { return len(s.a), len(s.b) }
func (s bytesSeqs) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenBytes(s.a[ai:aj:aj], s.b[bi:bj:bj])
}
func (s bytesSeqs) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenBytes(s.a[ai:aj:aj], s.b[bi:bj:bj])
}

type runesSeqs struct{ a, b []rune }

func (s runesSeqs) lengths() (int, int) { return len(s.a), len(s.b) }
func (s runesSeqs) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenRunes(s.a[ai:aj:aj], s.b[bi:bj:bj])
}
func (s runesSeqs) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenRunes(s.a[ai:aj:aj], s.b[bi:bj:bj])
}

// TODO(adonovan): optimize these functions using ideas from:
// - https://go.dev/cl/408116 common.go
// - https://go.dev/cl/421435 xor_generic.go

// TODO(adonovan): factor using generics when available,
// but measure performance impact.

// commonPrefixLen* returns the length of the common prefix of a[ai:aj] and b[bi:bj].
func commonPrefixLenBytes(a, b []byte) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}
func commonPrefixLenRunes(a, b []rune) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}
func commonPrefixLenString(a, b string) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}
func commonPrefixLenLines[S1, S2 text.String](a []S1, b []S2) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && text.Equal(a[i], b[i]) {
		i++
	}
	return i
}
func commonPrefixLenSlices[T comparable, S1 ~[]T, S2 ~[]T](a S1, b S2) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}
func commonPrefixLenAnySlices[T1, T2 any, S1 ~[]T1, S2 ~[]T2, C EqualsComparer[T1, T2]](a S1, b S2, c C) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && c.Equal(a[i], b[i]) {
		i++
	}
	return i
}

// commonSuffixLen* returns the length of the common suffix of a[ai:aj] and b[bi:bj].
func commonSuffixLenBytes(a, b []byte) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[len(a)-1-i] == b[len(b)-1-i] {
		i++
	}
	return i
}
func commonSuffixLenRunes(a, b []rune) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[len(a)-1-i] == b[len(b)-1-i] {
		i++
	}
	return i
}
func commonSuffixLenString(a, b string) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[len(a)-1-i] == b[len(b)-1-i] {
		i++
	}
	return i
}
func commonSuffixLenLines[S1, S2 text.String](a []S1, b []S2) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && text.Equal(a[len(a)-1-i], b[len(b)-1-i]) {
		i++
	}
	return i
}
func commonSuffixLenSlices[T comparable, S1 ~[]T, S2 ~[]T](a S1, b S2) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[len(a)-1-i] == b[len(b)-1-i] {
		i++
	}
	return i
}
func commonSuffixLenAnySlices[T1, T2 any, S1 ~[]T1, S2 ~[]T2, C EqualsComparer[T1, T2]](a S1, b S2, c C) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && c.Equal(a[len(a)-1-i], b[len(b)-1-i]) {
		i++
	}
	return i
}

func min(x, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}
