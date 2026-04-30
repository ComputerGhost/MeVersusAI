package bm25

import (
	"cmp"
	"log"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/clipperhouse/uax29/v2/words"
)

type Search struct {
	k1     float64 // Free parameter. See NewSearch for details.
	b      float64 // Free parameter. See NewSearch for details.
	corpus Corpus
}

// NewSearch creates a new BM25 algorithm context.
//
// k1 is used to approximate the saturation model via a parametric curve.
// With a high value, increases in term frequencies affect scores more.
// With a low value, increases in term frequencies affect scores less.
// For more corpora, a value between 1.2 and 2 is good.
//
// b affects the strength of document-length normalization.
// With a high value, it applies more; with a low value, it applies less.
// For more corpora, a value between 0.5 and 0.8 is good.
//
// Reference: https://www.staff.city.ac.uk/~sbrp622/papers/foundations_bm25_review.pdf
func NewSearch(k1 float64, b float64) *Search {
	if k1 < 0 || b < 0 || b > 1 {
		log.Fatalf("NewSearch: k1 %f, b %f", k1, b)
	}
	return &Search{k1: k1, b: b, corpus: NewCorpus()}
}

func (s *Search) Add(name, contents string) {
	tokens := Tokenize(contents)
	s.corpus.Add(name, NewDocument(tokens))
}

func (s *Search) Remove(name string) {
	s.corpus.Remove(name)
}

// Search returns document names sorted by their score for the query.
// The best match is listed first.
// The relative order of equal matches is undefined.
func (s *Search) Search(query string) []string {
	scores := s.Scores(query)

	names := make([]string, 0, len(scores))
	for name := range scores {
		names = append(names, name)
	}

	slices.SortFunc(names, func(a, b string) int {
		if c := cmp.Compare(scores[b], scores[a]); c != 0 {
			return c
		}
		return cmp.Compare(a, b) // deterministic fallback
	})

	return names
}

// Scores calculates the score of each document for the query.
// A document's score means nothing in isolation, but it can be used to compare
// documents for search result ranking. A higher value means a closer match.
func (s *Search) Scores(query string) map[string]float64 {
	documents := s.corpus.Documents()

	result := make(map[string]float64)
	for _, word := range Tokenize(query) {
		idf := s.idf(word)
		for name, doc := range documents {
			result[name] += idf * s.saturation(doc, word)
		}
	}
	return result
}

// The idf is the importance of the word based on its rarity.
func (s *Search) idf(word string) float64 {
	// A modified Robertson/Sparck Jones formula works well for this.
	docCount := float64(s.corpus.Size())
	docFrequency := float64(s.corpus.Count(word))
	return math.Log((docCount-docFrequency+0.5)/(docFrequency+0.5) + 1)
}

// The saturation is how much the word is used in the document.
func (s *Search) saturation(doc Document, word string) float64 {
	// Use the relative document size to normalize the value.
	avgDocSize := s.corpus.AverageDocumentSize()
	sizeWeight := 1 - s.b + s.b*float64(doc.Size())/avgDocSize

	// Return count/size ratio but with normalizations and diminishing returns.
	wordCount := float64(doc.Count(word))
	return wordCount * (s.k1 + 1) / (wordCount + s.k1*sizeWeight)
}

type Document struct {
	tokens []string
}

func NewDocument(tokens []string) Document {
	return Document{tokens}
}

func (d Document) Size() int {
	return len(d.tokens)
}

// Count returns the number of times a word appears in the document.
func (d Document) Count(word string) int {
	count := 0
	for _, t := range d.tokens {
		if t == word {
			count++
		}
	}
	return count
}

// Tokenize converts text into normalized search tokens.
//
// While the instructions only list English as a requirement,
// this should work for all romance languages.
// Full g11n, however, would require something much more complex.
func Tokenize(text string) []string {
	tokens := make([]string, 0)
	for iter := words.FromString(text); iter.Next(); {
		token := strings.ToLower(iter.Value())
		if token == "" || !containsAlphaNumRune(token) {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

// containsAlphaNumRune returns whether text contains an alphanumeric rune.
// This is useful to filter out noisy tokens like punctuation and emojis.
func containsAlphaNumRune(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

type Corpus struct {
	documents    map[string]Document
	wordCount    int            // Total word count across documents
	docsWithWord map[string]int // Number of documents that contain each word
}

func NewCorpus() Corpus {
	return Corpus{
		documents:    make(map[string]Document),
		docsWithWord: make(map[string]int),
	}
}

// Add processes and saves a document.
func (c *Corpus) Add(name string, document Document) {
	c.documents[name] = document
	c.wordCount += document.Size()
	for _, word := range removeDuplicates(document.tokens) {
		c.docsWithWord[word]++
	}
}

// Remove removes all data associated with a document.
func (c *Corpus) Remove(name string) {
	if doc, found := c.documents[name]; found {
		delete(c.documents, name)
		c.wordCount -= doc.Size()
		for _, word := range removeDuplicates(doc.tokens) {
			c.docsWithWord[word]--
		}
	}
}

func removeDuplicates(tokens []string) (result []string) {
	visited := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if _, ok := visited[token]; !ok {
			result = append(result, token)
			visited[token] = struct{}{}
		}
	}
	return result
}

func (c *Corpus) AverageDocumentSize() float64 {
	if docCount := len(c.documents); docCount > 0 {
		return float64(c.wordCount) / float64(docCount)
	}
	return 0
}

// Count returns the number of documents containing a word.
func (c *Corpus) Count(word string) int {
	return c.docsWithWord[word]
}

// Documents returns a map of document names to documents.
func (c *Corpus) Documents() map[string]Document {
	return c.documents
}

// Size returns the number of documents in the corpus.
func (c *Corpus) Size() int {
	return len(c.documents)
}
