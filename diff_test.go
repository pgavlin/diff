package gotextdiff_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	diff "github.com/pgavlin/gotextdiff"
	"github.com/pgavlin/gotextdiff/difftest"
	"github.com/pgavlin/gotextdiff/myers"
	"github.com/pgavlin/gotextdiff/span"
	"github.com/pgavlin/gotextdiff/text"
)

func TestApplyEdits(t *testing.T) {
	for _, tc := range difftest.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			if got := diff.ApplyEdits(tc.In, tc.Edits); got != tc.Out {
				t.Errorf("ApplyEdits edits got %q, want %q", got, tc.Out)
			}
			if tc.LineEdits != nil {
				if got := diff.ApplyEdits(tc.In, tc.LineEdits); got != tc.Out {
					t.Errorf("ApplyEdits lineEdits got %q, want %q", got, tc.Out)
				}
			}
		})
	}
}

func TestLineEdits(t *testing.T) {
	for _, tc := range difftest.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			// if line edits not specified, it is the same as edits
			edits := tc.LineEdits
			if edits == nil {
				edits = tc.Edits
			}
			if got := diff.LineEdits(tc.In, tc.Edits); diffEdits(got, edits) {
				t.Errorf("LineEdits got %q, want %q", got, edits)
			}
		})
	}
}

func TestUnified(t *testing.T) {
	for _, tc := range difftest.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			unified := fmt.Sprint(diff.ToUnified(difftest.FileA, difftest.FileB, tc.In, tc.Edits))
			if unified != tc.Unified {
				t.Errorf("edits got diff:\n%v\nexpected:\n%v", unified, tc.Unified)
			}
			if tc.LineEdits != nil {
				unified := fmt.Sprint(diff.ToUnified(difftest.FileA, difftest.FileB, tc.In, tc.LineEdits))
				if unified != tc.Unified {
					t.Errorf("lineEdits got diff:\n%v\nexpected:\n%v", unified, tc.Unified)
				}
			}
		})
	}
}

func diffEdits[T text.Text](got, want []diff.TextEdit[T]) bool {
	if len(got) != len(want) {
		return true
	}
	for i, w := range want {
		g := got[i]
		if span.Compare(w.Span, g.Span) != 0 {
			return true
		}
		if w.NewText.String() != g.NewText.String() {
			return true
		}
	}
	return false
}

func benchmarkDiff[T text.Text](b *testing.B, t1, t2 T) {
	b.Run("strings", func(b *testing.B) {
		benchmarkDiffCore(b, string(t1), string(t2))
	})

	b.Run("bytes", func(b *testing.B) {
		benchmarkDiffCore(b, []byte(t1), []byte(t2))
	})
}

func benchmarkDiffCore[T text.Text](b *testing.B, t1, t2 T) {
	b.Run("myers.ComputeEdits", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			myers.ComputeEdits("", t1, t2)
		}
	})

	b.Run("ApplyEdits", func(b *testing.B) {
		edits := myers.ComputeEdits("", t1, t2)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			diff.ApplyEdits(t1, edits)
		}
	})
}

func BenchmarkDiffUnrelated(b *testing.B) {
	s1 := "`Twas brillig, and the slithy toves\nDid gyre and gimble in the wabe:\nAll mimsy were the borogoves,\nAnd the mome raths outgrabe.\n"
	s2 := "I am the very model of a modern major general,\nI've information vegetable, animal, and mineral,\nI know the kings of England, and I quote the fights historical,\nFrom Marathon to Waterloo, in order categorical.\n"

	// Expand the text.
	for x := 0; x < 10; x++ {
		s1, s2 = s1+s1, s2+s2
	}

	benchmarkDiff(b, s1, s2)
}

func BenchmarkDiffJournalRegister(b *testing.B) {
	d1, err := ioutil.ReadFile("testdata/journal-register-base.txt")
	if err != nil {
		b.Fatalf("reading test data: %v", err)
	}
	d2, err := ioutil.ReadFile("testdata/journal-register-edit.txt")
	if err != nil {
		b.Fatalf("reading test data: %v", err)
	}

	benchmarkDiff(b, d1, d2)
}

func BenchmarkDiffGlagolitic(b *testing.B) {
	d1, err := ioutil.ReadFile("testdata/glagolitic-base.txt")
	if err != nil {
		b.Fatalf("reading test data: %v", err)
	}
	d2, err := ioutil.ReadFile("testdata/glagolitic-edit.txt")
	if err != nil {
		b.Fatalf("reading test data: %v", err)
	}

	benchmarkDiff(b, d1, d2)
}
