package bm25

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

type Engine struct {
	docs      map[string]*document
	docFreq   map[string]int
	totalLen  int
	avgDocLen float64
}

type document struct {
	name   string
	terms  map[string]int
	length int
}

func New() *Engine {
	return &Engine{
		docs:    make(map[string]*document),
		docFreq: make(map[string]int),
	}
}

func (e *Engine) Add(filename, contents string) {
	e.Remove(filename)

	terms := tokenize(contents)
	if len(terms) == 0 {
		e.docs[filename] = &document{
			name:  filename,
			terms: map[string]int{},
		}
		e.recalculateAverageLength()
		return
	}

	frequencies := make(map[string]int)
	for _, term := range terms {
		frequencies[term]++
	}

	doc := &document{
		name:   filename,
		terms:  frequencies,
		length: len(terms),
	}

	e.docs[filename] = doc
	e.totalLen += doc.length

	for term := range frequencies {
		e.docFreq[term]++
	}

	e.recalculateAverageLength()
}

func (e *Engine) Remove(filename string) {
	doc, ok := e.docs[filename]
	if !ok {
		return
	}

	delete(e.docs, filename)
	e.totalLen -= doc.length

	for term := range doc.terms {
		e.docFreq[term]--
		if e.docFreq[term] <= 0 {
			delete(e.docFreq, term)
		}
	}

	e.recalculateAverageLength()
}

func (e *Engine) Search(query string) []string {
	queryTerms := tokenize(query)
	if len(queryTerms) == 0 || len(e.docs) == 0 {
		return nil
	}

	queryFreq := make(map[string]int)
	for _, term := range queryTerms {
		queryFreq[term]++
	}

	type result struct {
		name  string
		score float64
	}

	results := make([]result, 0, len(e.docs))

	for _, doc := range e.docs {
		score := e.score(doc, queryFreq)
		if score > 0 {
			results = append(results, result{
				name:  doc.name,
				score: score,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].name < results[j].name
		}
		return results[i].score > results[j].score
	})

	names := make([]string, len(results))
	for i, result := range results {
		names[i] = result.name
	}

	return names
}

func (e *Engine) score(doc *document, queryFreq map[string]int) float64 {
	const k1 = 1.5
	const b = 0.75

	if doc.length == 0 || e.avgDocLen == 0 {
		return 0
	}

	var score float64
	docCount := float64(len(e.docs))

	for term, qf := range queryFreq {
		tf := float64(doc.terms[term])
		if tf == 0 {
			continue
		}

		df := float64(e.docFreq[term])
		idf := math.Log(1 + (docCount-df+0.5)/(df+0.5))

		numerator := tf * (k1 + 1)
		denominator := tf + k1*(1-b+b*float64(doc.length)/e.avgDocLen)

		score += float64(qf) * idf * numerator / denominator
	}

	return score
}

func (e *Engine) recalculateAverageLength() {
	if len(e.docs) == 0 {
		e.avgDocLen = 0
		return
	}

	e.avgDocLen = float64(e.totalLen) / float64(len(e.docs))
}

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func tokenize(text string) []string {
	text = strings.ToLower(text)
	return tokenPattern.FindAllString(text, -1)
}
