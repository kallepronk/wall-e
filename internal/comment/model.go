package comment

type Comment struct {
	FilePath  string
	Text      string
	Line      int
	StartByte uint32
	EndByte   uint32
}
