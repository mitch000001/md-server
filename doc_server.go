package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/russross/blackfriday"
	"github.com/shurcooL/go/github_flavored_markdown"
)

var dirFlag = flag.String("dir", "", "The dir to work on")
var offlineFlag = flag.Bool("offline", true, "Run offline (default: true)")

var cleanupFuncs []func()

func main() {
	flag.Parse()
	dir := *dirFlag
	if dir == "" {
		dir, _ = os.Getwd()
	}
	defer func() {
		fmt.Printf("Cleanup: %+#v\n", cleanupFuncs)
		for _, fn := range cleanupFuncs {
			fmt.Printf("Cleanup")
			fn()
		}
	}()
	http.Handle("/", http.FileServer(&fileSystem{http.Dir(*dirFlag)}))
	fmt.Printf("Listening on localhost:2000\n")
	log.Fatal(http.ListenAndServe(":2000", nil))
}

type file struct {
	underlyingFile http.File
	*bytes.Reader
}

func (f *file) Readdir(n int) (fi []os.FileInfo, err error) {
	return f.underlyingFile.Readdir(n)
}

func (f *file) Stat() (fi os.FileInfo, err error) {
	fileStat, err := f.underlyingFile.Stat()
	if err != nil {
		return nil, err
	}
	return &fileInfo{FileInfo: fileStat, reader: f.Reader}, nil
}

func newFile(f http.File, fBytes []byte) *file {
	htmlFile := &file{underlyingFile: f}
	mdBytes := githubFlavouredMarkdown(fBytes)
	htmlFile.Reader = bytes.NewReader(mdBytes)
	return htmlFile
}

type fileInfo struct {
	os.FileInfo
	reader *bytes.Reader
}

func (fi *fileInfo) Size() int64 {
	return int64(fi.reader.Len())
}

type fileSystem struct {
	http.Dir
}

func (fs *fileSystem) Open(name string) (http.File, error) {
	fmt.Printf("Open: %s\n", name)
	f, err := fs.Dir.Open(name)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return f, nil
	}
	fileBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return newFile(f, fileBytes), nil
}

// Close consumes the Reader so that the next calls to Read will return an eof error
func (f *file) Close() error {
	return f.underlyingFile.Close()
}

var commonHtmlFlags = 0 |
	blackfriday.HTML_USE_XHTML |
	blackfriday.HTML_COMPLETE_PAGE |
	blackfriday.HTML_USE_SMARTYPANTS |
	blackfriday.HTML_SMARTYPANTS_FRACTIONS |
	blackfriday.HTML_SMARTYPANTS_LATEX_DASHES

var extensions = 0 |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_HEADER_IDS

type markdownProvider interface {
	Markdown([]byte) []byte
}

type markdownFunc func([]byte) []byte

func (m markdownFunc) Markdown(in []byte) []byte {
	return (in)
}

func newLocalGithubFlavouredMarkdown() markdownProvider {
	return markdownFunc(github_flavored_markdown.Markdown)
}

func githubFlavouredMarkdown(in []byte) []byte {
	var buf bytes.Buffer
	if *offlineFlag {
		fn := WriteGitHubFlavoredMarkdownViaLocal(&buf, in, *dirFlag)
		cleanupFuncs = append(cleanupFuncs, fn)
	} else {
		WriteGitHubFlavoredMarkdownViaGitHub(&buf, in)
	}
	return buf.Bytes()
}

type customBlackfridayMarkdown struct {
	htmlFlags          int
	markdownExtensions int
}

func (c *customBlackfridayMarkdown) Markdown(in []byte) []byte {
	return blackfriday.Markdown(in, blackfriday.HtmlRenderer(c.htmlFlags, "", ""), c.markdownExtensions)
}

func newBlackfridayMarkdown() markdownProvider {
	c := &customBlackfridayMarkdown{
		htmlFlags:          commonHtmlFlags,
		markdownExtensions: extensions,
	}
	return c
}

func blackfridayMarkdown(in []byte) []byte {
	return newBlackfridayMarkdown().Markdown(in)
}

func indexHandler(rootPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="/test">test</a>`))
	}
}

type indexWalker struct {
	dirToFiles map[string][]string
}

func (i *indexWalker) walk(path string, info os.FileInfo, err error) error {
	if i.dirToFiles == nil {
		i.dirToFiles = make(map[string][]string)
	}
	if info.IsDir() {
		if _, ok := i.dirToFiles[path]; !ok {
			i.dirToFiles[path] = make([]string, 0)
		}
	}
	if !info.IsDir() {
		i.dirToFiles[filepath.Dir(path)] = append(i.dirToFiles[filepath.Dir(path)], info.Name())
	}
	return nil
}
