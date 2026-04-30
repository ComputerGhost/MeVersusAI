package bm25

type BM25 interface {
	// Add adds a file to the corpus.
	Add(filename, contents string)

	// Remove removes a file from the corpus.
	Remove(filename string)

	// Search returns the names of all documents ranked from best match to worst match.
	Search(query string) []string
}
