package diff_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/pgavlin/diff"
	"github.com/pgavlin/diff/myers"
	"github.com/pgavlin/text"
)

func diffSize[T text.String](t testing.TB, edits []diff.Edit[T]) int {
	type editJSON struct {
		Start int    `json:"start,omitempty"`
		End   int    `json:"end,omitempty"`
		New   string `json:"new,omitempty"`
	}

	editsJSON := make([]editJSON, len(edits))
	for i, e := range edits {
		editsJSON[i] = editJSON{Start: e.Start, End: e.End, New: string(e.New)}
	}

	bytes, err := json.Marshal(editsJSON)
	if err != nil {
		t.Fatalf("marshaling edits: %v", err)
	}

	return len(bytes)
}

func benchmarkDiff[T text.String](b *testing.B, t1, t2 T) {
	b.Run("strings", func(b *testing.B) {
		benchmarkDiffCore(b, string(t1), string(t2))
	})

	b.Run("bytes", func(b *testing.B) {
		benchmarkDiffCore(b, []byte(t1), []byte(t2))
	})
}

func benchmarkDiffCore[T text.String](b *testing.B, t1, t2 T) {
	b.Run("myers", func(b *testing.B) {
		b.Run("ComputeEdits", func(b *testing.B) {
			b.ReportMetric(0, "bytes")
			for i := 0; i < b.N; i++ {
				myers.ComputeEdits(t1, t2)
			}
		})
		b.Run("Apply", func(b *testing.B) {
			edits := myers.ComputeEdits(t1, t2)
			b.ResetTimer()
			b.ReportMetric(float64(diffSize(b, edits)), "bytes")

			for i := 0; i < b.N; i++ {
				diff.Apply(t1, edits)
			}
		})
	})

	b.Run("lcs", func(b *testing.B) {
		b.Run("Text", func(b *testing.B) {
			b.ReportMetric(0, "bytes")
			for i := 0; i < b.N; i++ {
				diff.Text(t1, t2)
			}
		})

		b.Run("Binary", func(b *testing.B) {
			b.ReportMetric(0, "bytes")
			for i := 0; i < b.N; i++ {
				diff.Binary(t1, t2)
			}
		})

		b.Run("Apply", func(b *testing.B) {
			edits := diff.Text(t1, t2)
			b.ResetTimer()
			b.ReportMetric(float64(diffSize(b, edits)), "bytes")

			for i := 0; i < b.N; i++ {
				diff.Apply(t1, edits)
			}
		})
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
