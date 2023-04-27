// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package difftest supplies a set of tests that will operate on any
// implementation of a diff algorithm as exposed by
// "github.com/pgavlin/gotextdiff"
package difftest

import (
	"fmt"
	"testing"

	diff "github.com/pgavlin/gotextdiff"
	"github.com/pgavlin/gotextdiff/span"
	"github.com/pgavlin/gotextdiff/text"
)

const (
	FileA         = "from"
	FileB         = "to"
	UnifiedPrefix = "--- " + FileA + "\n+++ " + FileB + "\n"
)

var TestCases = []struct {
	Name, In, Out, Unified string
	Edits, LineEdits       []diff.TextEdit[string]
	NoDiff                 bool
}{{
	Name: "empty",
	In:   "",
	Out:  "",
}, {
	Name: "no_diff",
	In:   "gargantuan\n",
	Out:  "gargantuan\n",
}, {
	Name: "replace_all",
	In:   "fruit\n",
	Out:  "cheese\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-fruit
+cheese
`[1:],
	Edits:     []diff.TextEdit[string]{{Span: newSpan(0, 5), NewText: diff.NewRope("cheese")}},
	LineEdits: []diff.TextEdit[string]{{Span: newSpan(0, 6), NewText: diff.NewRope("cheese\n")}},
}, {
	Name: "insert_rune",
	In:   "gord\n",
	Out:  "gourd\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-gord
+gourd
`[1:],
	Edits:     []diff.TextEdit[string]{{Span: newSpan(2, 2), NewText: diff.NewRope("u")}},
	LineEdits: []diff.TextEdit[string]{{Span: newSpan(0, 5), NewText: diff.NewRope("gourd\n")}},
}, {
	Name: "delete_rune",
	In:   "groat\n",
	Out:  "goat\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-groat
+goat
`[1:],
	Edits:     []diff.TextEdit[string]{{Span: newSpan(1, 2), NewText: diff.NewRope("")}},
	LineEdits: []diff.TextEdit[string]{{Span: newSpan(0, 6), NewText: diff.NewRope("goat\n")}},
}, {
	Name: "replace_rune",
	In:   "loud\n",
	Out:  "lord\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-loud
+lord
`[1:],
	Edits:     []diff.TextEdit[string]{{Span: newSpan(2, 3), NewText: diff.NewRope("r")}},
	LineEdits: []diff.TextEdit[string]{{Span: newSpan(0, 5), NewText: diff.NewRope("lord\n")}},
}, {
	Name: "replace_partials",
	In:   "blanket\n",
	Out:  "bunker\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-blanket
+bunker
`[1:],
	Edits: []diff.TextEdit[string]{
		{Span: newSpan(1, 3), NewText: diff.NewRope("u")},
		{Span: newSpan(6, 7), NewText: diff.NewRope("r")},
	},
	LineEdits: []diff.TextEdit[string]{{Span: newSpan(0, 8), NewText: diff.NewRope("bunker\n")}},
}, {
	Name: "insert_line",
	In:   "1: one\n3: three\n",
	Out:  "1: one\n2: two\n3: three\n",
	Unified: UnifiedPrefix + `
@@ -1,2 +1,3 @@
 1: one
+2: two
 3: three
`[1:],
	Edits: []diff.TextEdit[string]{{Span: newSpan(7, 7), NewText: diff.NewRope("2: two\n")}},
}, {
	Name: "replace_no_newline",
	In:   "A",
	Out:  "B",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-A
\ No newline at end of file
+B
\ No newline at end of file
`[1:],
	Edits: []diff.TextEdit[string]{{Span: newSpan(0, 1), NewText: diff.NewRope("B")}},
}, {
	Name: "add_end",
	In:   "A",
	Out:  "AB",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-A
\ No newline at end of file
+AB
\ No newline at end of file
`[1:],
	Edits:     []diff.TextEdit[string]{{Span: newSpan(1, 1), NewText: diff.NewRope("B")}},
	LineEdits: []diff.TextEdit[string]{{Span: newSpan(0, 1), NewText: diff.NewRope("AB")}},
}, {
	Name: "add_newline",
	In:   "A",
	Out:  "A\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-A
\ No newline at end of file
+A
`[1:],
	Edits:     []diff.TextEdit[string]{{Span: newSpan(1, 1), NewText: diff.NewRope("\n")}},
	LineEdits: []diff.TextEdit[string]{{Span: newSpan(0, 1), NewText: diff.NewRope("A\n")}},
}, {
	Name: "delete_front",
	In:   "A\nB\nC\nA\nB\nB\nA\n",
	Out:  "C\nB\nA\nB\nA\nC\n",
	Unified: UnifiedPrefix + `
@@ -1,7 +1,6 @@
-A
-B
 C
+B
 A
 B
-B
 A
+C
`[1:],
	Edits: []diff.TextEdit[string]{
		{Span: newSpan(0, 4), NewText: diff.NewRope("")},
		{Span: newSpan(6, 6), NewText: diff.NewRope("B\n")},
		{Span: newSpan(10, 12), NewText: diff.NewRope("")},
		{Span: newSpan(14, 14), NewText: diff.NewRope("C\n")},
	},
	NoDiff: true, // diff algorithm produces different delete/insert pattern
},
	{
		Name: "replace_last_line",
		In:   "A\nB\n",
		Out:  "A\nC\n\n",
		Unified: UnifiedPrefix + `
@@ -1,2 +1,3 @@
 A
-B
+C
+
`[1:],
		Edits:     []diff.TextEdit[string]{{Span: newSpan(2, 3), NewText: diff.NewRope("C\n")}},
		LineEdits: []diff.TextEdit[string]{{Span: newSpan(2, 4), NewText: diff.NewRope("C\n\n")}},
	},
	{
		Name: "multiple_replace",
		In:   "A\nB\nC\nD\nE\nF\nG\n",
		Out:  "A\nH\nI\nJ\nE\nF\nK\n",
		Unified: UnifiedPrefix + `
@@ -1,7 +1,7 @@
 A
-B
-C
-D
+H
+I
+J
 E
 F
-G
+K
`[1:],
		Edits: []diff.TextEdit[string]{
			{Span: newSpan(2, 8), NewText: diff.NewRope("H\nI\nJ\n")},
			{Span: newSpan(12, 14), NewText: diff.NewRope("K\n")},
		},
		NoDiff: true, // diff algorithm produces different delete/insert pattern
	},
}

func init() {
	// expand all the spans to full versions
	// we need them all to have their line number and column
	for _, tc := range TestCases {
		c := span.NewContentConverter("", []byte(tc.In))
		for i := range tc.Edits {
			tc.Edits[i].Span, _ = tc.Edits[i].Span.WithAll(c)
		}
		for i := range tc.LineEdits {
			tc.LineEdits[i].Span, _ = tc.LineEdits[i].Span.WithAll(c)
		}
	}
}

func DiffTest[T text.Text](t *testing.T, compute diff.ComputeEdits[T]) {
	t.Helper()
	for _, test := range TestCases {
		t.Run(test.Name, func(t *testing.T) {
			t.Helper()
			edits := compute(span.URIFromPath("/"+test.Name), T(test.In), T(test.Out))
			got := string(diff.ApplyEdits(T(test.In), edits))
			unified := fmt.Sprint(diff.ToUnified(FileA, FileB, T(test.In), edits))
			if got != string(test.Out) {
				t.Errorf("got patched:\n%v\nfrom diff:\n%v\nexpected:\n%v", got, unified, string(test.Out))
			}
			if !test.NoDiff && unified != string(test.Unified) {
				t.Errorf("got diff:\n%v\nexpected:\n%v", unified, string(test.Unified))
			}
		})
	}
}

func newSpan(start, end int) span.Span {
	return span.New("", span.NewPoint(0, 0, start), span.NewPoint(0, 0, end))
}
