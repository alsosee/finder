package main

import "github.com/alsosee/finder/structs"

func cloneContents(contents structs.Contents) structs.Contents {
	cloned := make(structs.Contents, len(contents))
	for id, content := range contents {
		cloned[id] = content
	}
	return cloned
}
