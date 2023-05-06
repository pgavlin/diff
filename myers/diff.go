// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package myers implements the Myers diff algorithm.
package myers

import (
	"github.com/pgavlin/diff"
	"github.com/pgavlin/text"
)

// Sources:
// https://blog.jcoglan.com/2017/02/17/the-myers-diff-algorithm-part-3/
// https://www.codeproject.com/Articles/42279/%2FArticles%2F42279%2FInvestigating-Myers-diff-algorithm-Part-1-of-2

type Operation struct {
	Kind               diff.OpKind
	Start, End         int // indices of the line in a
	ReplStart, ReplEnd int // indices of the line in b, J2 implied by len(Content)
}

func ComputeEdits[S1, S2 text.String](before S1, after S2) []diff.Edit[S2] {
	beforeLines, afterLines := splitLines(before), splitLines(after)
	ops := Operations(beforeLines, afterLines)

	// Build a table mapping line number to offset.
	lineOffsets := make([]int, 0, len(beforeLines)+1)
	total := 0
	for i := range beforeLines {
		lineOffsets = append(lineOffsets, total)
		total += len(beforeLines[i])
	}
	lineOffsets = append(lineOffsets, total) // EOF

	edits := make([]diff.Edit[S2], 0, len(ops))
	for _, op := range ops {
		start, end := lineOffsets[op.Start], lineOffsets[op.End]
		switch op.Kind {
		case diff.Delete:
			// Delete: before[I1:I2] is deleted.
			edits = append(edits, diff.Edit[S2]{Start: start, End: end})
		case diff.Insert:
			// Insert: after[J1:J2] is inserted at before[I1:I1].
			if content := text.Join(afterLines[op.ReplStart:op.ReplEnd], ""); len(content) != 0 {
				edits = append(edits, diff.Edit[S2]{Start: start, End: end, New: content})
			}
		}
	}
	return edits
}

// Operations returns the list of operations to convert a into b, consolidating
// operations for multiple lines and not including equal lines.
func Operations[S1, S2 text.String, A ~[]S1, B ~[]S2](a A, b B) []Operation {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}

	trace, offset := shortestEditSequence(a, b)
	snakes := backtrack(trace, len(a), len(b), offset)

	M, N := len(a), len(b)

	var i int
	solution := make([]Operation, len(a)+len(b))

	add := func(op *Operation, i2, j2 int) {
		if op == nil {
			return
		}
		op.End = i2
		if op.Kind == diff.Insert {
			op.ReplEnd = j2
		}
		solution[i] = *op
		i++
	}
	x, y := 0, 0
	for _, snake := range snakes {
		if len(snake) < 2 {
			continue
		}
		var op *Operation
		// delete (horizontal)
		for snake[0]-snake[1] > x-y {
			if op == nil {
				op = &Operation{
					Kind:      diff.Delete,
					Start:     x,
					ReplStart: y,
				}
			}
			x++
			if x == M {
				break
			}
		}
		add(op, x, y)
		op = nil
		// insert (vertical)
		for snake[0]-snake[1] < x-y {
			if op == nil {
				op = &Operation{
					Kind:      diff.Insert,
					Start:     x,
					ReplStart: y,
				}
			}
			y++
		}
		add(op, x, y)
		op = nil
		// equal (diagonal)
		for x < snake[0] {
			x++
			y++
		}
		if x >= M && y >= N {
			break
		}
	}
	return solution[:i]
}

// backtrack uses the trace for the edit sequence computation and returns the
// "snakes" that make up the solution. A "snake" is a single deletion or
// insertion followed by zero or diagonals.
func backtrack(trace [][]int, x, y, offset int) [][]int {
	snakes := make([][]int, len(trace))
	d := len(trace) - 1
	for ; x > 0 && y > 0 && d > 0; d-- {
		V := trace[d]
		if len(V) == 0 {
			continue
		}
		snakes[d] = []int{x, y}

		k := x - y

		var kPrev int
		if k == -d || (k != d && V[k-1+offset] < V[k+1+offset]) {
			kPrev = k + 1
		} else {
			kPrev = k - 1
		}

		x = V[kPrev+offset]
		y = x - kPrev
	}
	if x < 0 || y < 0 {
		return snakes
	}
	snakes[d] = []int{x, y}
	return snakes
}

// shortestEditSequence returns the shortest edit sequence that converts a into b.
func shortestEditSequence[S1, S2 text.String](a []S1, b []S2) ([][]int, int) {
	M, N := len(a), len(b)
	V := make([]int, 2*(N+M)+1)
	offset := N + M
	trace := make([][]int, N+M+1)

	// Iterate through the maximum possible length of the SES (N+M).
	for d := 0; d <= N+M; d++ {
		copyV := make([]int, len(V))
		// k lines are represented by the equation y = x - k. We move in
		// increments of 2 because end points for even d are on even k lines.
		for k := -d; k <= d; k += 2 {
			// At each point, we either go down or to the right. We go down if
			// k == -d, and we go to the right if k == d. We also prioritize
			// the maximum x value, because we prefer deletions to insertions.
			var x int
			if k == -d || (k != d && V[k-1+offset] < V[k+1+offset]) {
				x = V[k+1+offset] // down
			} else {
				x = V[k-1+offset] + 1 // right
			}

			y := x - k

			// Diagonal moves while we have equal contents.
			for x < M && y < N && text.Equal(a[x], b[y]) {
				x++
				y++
			}

			V[k+offset] = x

			// Return if we've exceeded the maximum values.
			if x == M && y == N {
				// Makes sure to save the state of the array before returning.
				copy(copyV, V)
				trace[d] = copyV
				return trace, offset
			}
		}

		// Save the state of the array.
		copy(copyV, V)
		trace[d] = copyV
	}
	return nil, 0
}

func splitLines[S text.String](t S) []S {
	lines := text.SplitAfter(t, "\n")
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}
