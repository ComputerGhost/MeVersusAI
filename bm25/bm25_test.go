package bm25

import (
	"flag"
	"log"
	"os"
	"slices"
	"strings"
	"testing"
)

var factory func() BM25
var target = flag.String("target", "", "which implementation (ai|human) to test")

func TestMain(m *testing.M) {
	flag.Parse()

	switch *target {
	case "ai":
		factory = func() BM25 { return New() }
	case "human":
		factory = func() BM25 { return NewSearch(1.2, 0.75) }
	default:
		panic("invalid target: " + *target)
	}

	os.Exit(m.Run())
}

func TestBM25(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		corpus []string
		query  string
		rank   []string
	}{
		{
			name:   "empty",
			corpus: []string{},
			query:  "test",
			rank:   []string{},
		},
		{
			name:   "single match",
			corpus: []string{"blue", "test", "blue tulips"},
			query:  "test",
			rank:   []string{"test", "blue", "blue tulips"},
		},
		{
			name:   "multiple matches",
			corpus: []string{"blue", "test", "blue tulips"},
			query:  "blue",
			rank:   []string{"blue", "blue tulips", "test"},
		},
		{
			name:   "overused word",
			corpus: []string{"blue", "test", "test test"},
			query:  "test",
			rank:   []string{"test test", "test", "blue"},
		},
		{
			name:   "multiword query",
			corpus: []string{"test", "blue", "blue tulips"},
			query:  "tulips blue",
			rank:   []string{"blue tulips", "blue", "test"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bm := factory()
			for _, contents := range tt.corpus {
				filename := contents // just name the file after the contents
				bm.Add(filename, contents)
			}

			rank := bm.Search(tt.query)

			if !slices.Equal(rank, tt.rank) {
				want := formatCorpus(tt.rank)
				got := formatCorpus(rank)
				log.Fatalf("rank: expected %v, got %v", want, got)
			}
		})
	}
}

func formatCorpus(contents []string) string {
	return "[" + strings.Join(contents, ", ") + "]"
}
