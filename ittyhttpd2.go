package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	ENTRY_TYPE_DIR     = 0 // These can't use `iota` because the template needs to know them
	ENTRY_TYPE_REGULAR = 1
	ENTRY_TYPE_SYMLINK = 2
	ENTRY_TYPE_UNKNOWN = 3
)

var IGNORED_REQUEST_PATHS = [...]string{"favicon.ico"}

type EntryType uint8

type FileEntry struct {
	Name         string
	AbsolutePath string
	Type         EntryType
	Size         string
}

var templates *template.Template

var templateIndexHTML = `
<html>
<head>
</head>
<body>

index of {{.Base}}

<table>
{{ range $entry := .Entries }}
	<tr>
		{{ if eq $entry.Type 0 }}				<!--dir-->
			<td></td>
			<td><a href="{{$.Base}}{{ $entry.Name }}">{{ $entry.Name }}/</a></td>
		{{ else if eq $entry.Type 1 }}	<!--regular file-->
			<td>{{ $entry.Size }}</td>
			<td><a href="{{$.Base}}{{ $entry.Name }}">{{ $entry.Name }}</a></td>
		{{ else if eq $entry.Type 2 }}	<!--symlink-->
			<td></td>
			<td><a href="{{$.Base}}{{ $entry.Name }}">{{ $entry.Name }} →</a></td>
		{{ else }}
			<td></td>
			<td>{{ $entry.Name }} [unknown type {{ $entry.Type }}]</td>
		{{ end }}
	</tr>
{{ end }}
</table>

</body>
</html>
`

func handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	if shouldIgnoreRequest(path) {
		return
	}

	fmt.Printf("request \"%s\"\n", path)

	fileEntry, err := getFileInfo(path)
	if err != nil {
		http.Error(w, "getFileInfo failed for requested path", 403)
		return
	}

	if fileEntry.Type == ENTRY_TYPE_DIR {
		showIndex(w, path, fileEntry)
	} else if fileEntry.Type == ENTRY_TYPE_REGULAR {
		serveFile(w, r, fileEntry.AbsolutePath)
	} else {
		http.Error(
			w,
			fmt.Sprintf("Not a directory or regular file, so not accepting requested path \"%s\"\n", path),
			403,
		)
		return
	}
}

func shouldIgnoreRequest(path string) bool {
	for _, ignoredPath := range IGNORED_REQUEST_PATHS {
		if path == ignoredPath {
			return true
		}
	}
	return false
}

func getFileInfo(path string) (*FileEntry, error) {
	absolutePath := findAbsolutePath(path)
	fileInfo, err := os.Stat(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("Could not stat \"%s\": %s\n", absolutePath, err)
	}
	return buildFileEntry(fileInfo, absolutePath), nil
}

func buildFileEntry(fileInfo os.FileInfo, absolutePath string) *FileEntry {
	return &FileEntry{
		Name:         fileInfo.Name(),
		AbsolutePath: absolutePath,
		Type:         getFileEntryType(fileInfo),
		Size:         humanize.Bytes(uint64(fileInfo.Size())),
	}
}

func getFileEntryType(info os.FileInfo) EntryType {
	if info.Mode().IsDir() {
		return ENTRY_TYPE_DIR
	} else if info.Mode().IsRegular() {
		return ENTRY_TYPE_REGULAR
	} else if (info.Mode() & os.ModeSymlink) != 0 {
		return ENTRY_TYPE_SYMLINK
	}
	return ENTRY_TYPE_UNKNOWN
}

func showIndex(w http.ResponseWriter, path string, fileEntry *FileEntry) {
	var base string
	if path == "" {
		base = "/"
	} else {
		base = fmt.Sprintf("/%s/", path)
	}

	templateData := struct {
		Base    string
		Entries []FileEntry
	}{
		base,
		findFileEntriesForDir(fileEntry.AbsolutePath),
	}
	err := templates.ExecuteTemplate(w, "index.html", templateData)
	if err != nil {
		fmt.Println(err)
	}
}

func serveFile(w http.ResponseWriter, r *http.Request, absolutePath string) {
	if checkPathIsClean(absolutePath) {
		http.ServeFile(w, r, absolutePath)
	} else {
		http.Error(w, "Requested path is not allowed", 403)
	}
}

func findFileEntriesForDir(path string) []FileEntry {
	var fileEntries []FileEntry

	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Printf("Could not get directory list for \"%s\": %s\n", path, err)
	}

	for _, fileInfo := range fileInfos {
		absPath := findAbsolutePath(fileInfo.Name())
		fileEntry := buildFileEntry(fileInfo, absPath)
		if fileEntry.Type == ENTRY_TYPE_UNKNOWN {
			continue // don't list anything that isn't a directory, regular file, or symlink
		}
		fileEntries = append(fileEntries, *fileEntry)
	}

	return fileEntries
}

func checkPathIsClean(path string) bool {
	return !strings.Contains(path, "../")
}

func findAbsolutePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Could not find absolute path of requested directory: %s\n", err)
	}
	return absPath
}

func setupChdir(path string) {
	err := syscall.Chdir(path)
	if err != nil {
		fmt.Printf("Could not change to requested directory \"%s\": %s\n", path, err)
		os.Exit(4)
	} else {
		fmt.Printf("Changed directory to \"%s\"\n", path)
	}
}

func exitWithHelp() {
	fmt.Printf("This program takes exactly two arguments: %s PORT ROOT_DIR\n", os.Args[0])
	os.Exit(1)
}

func main() {
	templates = template.Must(template.New("index.html").Parse(templateIndexHTML))

	var rootPath string
	var port uint

	if len(os.Args) != 3 {
		exitWithHelp()
	}

	parsedCount, _ := fmt.Sscanf(os.Args[1], "%d", &port)
	if parsedCount != 1 {
		exitWithHelp()
	}

	parsedCount, _ = fmt.Sscanf(os.Args[2], "%s", &rootPath)
	if parsedCount != 1 {
		exitWithHelp()
	}

	if rootPath = findAbsolutePath(rootPath); rootPath != "" {
		setupChdir(rootPath)
	}

	http.HandleFunc("/", handleRequest)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
