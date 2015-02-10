package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/shurcooL/go/github_flavored_markdown"
)

var remoteCss = []string{
	"https://dl.dropboxusercontent.com/u/8554242/temp/github-flavored-markdown.css",
	"//cdnjs.cloudflare.com/ajax/libs/octicons/2.1.2/octicons.css",
}

var localCss = []string{
	"github-flavored-markdown.css",
	"octicons.css",
}

const styleLinkTemplate = `<link href="%s" media="all" rel="stylesheet" type="text/css" />`

// Convert GitHub Flavored Markdown to full HTML page and write to w.
func WriteGitHubFlavoredMarkdownViaLocal(w io.Writer, markdown []byte, root string) (cleanup func()) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		panic(err)
	}
	paths := createAssets(absRoot)
	io.WriteString(w, `<html><head><meta charset="utf-8">`)
	for _, link := range localCss {
		io.WriteString(w, styleLink(path.Clean("/"+link)))
	}
	io.WriteString(w, `</head><body><article class="markdown-body entry-content" style="padding: 30px;">`)
	w.Write(github_flavored_markdown.Markdown(markdown))
	io.WriteString(w, `</article></body></html>`)
	return func() {
		var errors []error
		for _, link := range paths {
			err := os.Remove(link)
			if err != nil {
				errors = append(errors, err)
			}
		}
		if len(errors) != 0 {
			panic(errors)
		}
	}
}

func createAssets(root string) []string {
	var paths []string
	for _, a := range AssetNames() {
		path := filepath.Join(root, a)
		fmt.Printf("Path: %s\n", path)
		data, err := Asset(a)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(path, data, 0700)
		if err != nil {
			panic(err)
		}
		paths = append(paths, path)
	}
	return paths
}

// TODO: Remove once local version gives matching results.
func WriteGitHubFlavoredMarkdownViaGitHub(w io.Writer, markdown []byte) {
	io.WriteString(w, `<html><head><meta charset="utf-8">`)
	for _, link := range remoteCss {
		io.WriteString(w, styleLink(link))
	}
	io.WriteString(w, `</head><body><article class="markdown-body entry-content" style="padding: 30px;">`)

	// Convert GitHub-Flavored-Markdown to HTML (includes syntax highlighting for diff, Go, etc.)
	resp, err := http.Post("https://api.github.com/markdown/raw", "text/x-markdown", bytes.NewReader(markdown))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		panic(err)
	}

	io.WriteString(w, `</article></body></html>`)
}

func styleLink(url string) string {
	return fmt.Sprintf(styleLinkTemplate, url)
}
