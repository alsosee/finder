package structs

type Processor interface {
	Init() error
	ProcessFiles(Contents) error
	ProcessDirectories(dirs map[string][]File) error
	Finalize() error
}

type ProcessorTodo interface {
	ProcessContent(content Content) error
}
