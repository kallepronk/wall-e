package scanner

type LanguageScanner interface {
	Scan(content []byte) ([]Comment, error)
}
