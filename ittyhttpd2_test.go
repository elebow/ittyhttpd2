package main

import (
	"fmt"
	"os"
	"testing"
)

func TestCheckPathIsClean(t *testing.T) {
	if !checkPathIsClean("safe") {
		t.Error("'safe' considered not clean")
	}

	if !checkPathIsClean("safe..txt") {
		t.Error("'safe.txt' considered not clean")
	}

	if checkPathIsClean("../unsafe") {
		t.Error("'../unsafe' considered clean")
	}

	if checkPathIsClean("/../unsafe") {
		t.Error("'/../unsafe' considered clean")
	}
}

func TestFindFileEntriesForDir(t *testing.T) {
	fileEntries := findFileEntriesForDir(".")

	found := false
	cwd, _ := os.Getwd()
	absPath := fmt.Sprintf("%s/ittyhttpd2_test.go", cwd)
	for i := range fileEntries {
		if fileEntries[i].Name == "ittyhttpd2_test.go" &&
			fileEntries[i].AbsolutePath == absPath {
			found = true
		}
	}
	if !found {
		t.Error("Couldn't find 'ittyhttpd2_test.go' in current directory.")
	}
}
