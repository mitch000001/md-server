package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEntryPoint(t *testing.T) {
	testPath := filepath.Join("/tmp", "doc-server")
	testFilePath := filepath.Join(testPath, "test.md")
	os.Mkdir(testPath, 0700)
	defer os.RemoveAll(testPath)

	testMarkdown := `
	# Header

	Text
	`
	testFile, _ := os.Create(testFilePath)
	testFile.WriteString(testMarkdown)
	testFile.Close()
}

func TestIndexHandler(t *testing.T) {
	testPath := filepath.Join("/tmp", "doc-server")
	testFileName := "test.md"
	testMarkdown := `
	# Header

	Text
	`
	data := testData{
		root:     testPath,
		filename: testFileName,
		content:  testMarkdown,
	}
	setupTestData(data)
	defer os.RemoveAll(testPath)

	request, _ := http.NewRequest("GET", "http://localhost/", nil)

	recorder := httptest.NewRecorder()
	indexHandler := indexHandler("/")
	if indexHandler == nil {
		t.Logf("Expected indexHandler not to be nil\n")
		t.FailNow()
	}
	indexHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Logf("Expected statuscode to be 200, got: %d\n", recorder.Code)
		t.Fail()
	}

	expectedHtml := `<a href="/test">test</a>`
	responseBody := recorder.Body.String()

	if !strings.Contains(responseBody, expectedHtml) {
		t.Logf("Expected '\n%s'\n to contain html: '%s'\n", responseBody, expectedHtml)
		t.Fail()
	}

	// Nested dirs

}

func TestIndexWalker(t *testing.T) {
	testPath := filepath.Join("/tmp", "doc-server")
	os.Mkdir(testPath, 0700)
	defer os.RemoveAll(testPath)
	testFiles := make([]string, 10)
	for i := 0; i < 10; i++ {
		testFileName := fmt.Sprintf("test_%d.md", i)
		testFiles[i] = testFileName
		os.Create(filepath.Join(testPath, testFileName))
	}
	walker := &indexWalker{}
	filepath.Walk(testPath, walker.walk)

	if len(walker.dirToFiles) != 1 {
		t.Logf("Expected only one entry, got %d\n", len(walker.dirToFiles))
		t.Fail()
	}
	expected := map[string][]string{testPath: testFiles}

	if !reflect.DeepEqual(expected, walker.dirToFiles) {
		t.Logf("Expected map to equal '%+#v', got: '%+#v'\n", expected, walker.dirToFiles)
		t.Fail()
	}

	// Nested directories
	nestedDir := filepath.Join(testPath, "nested")
	os.Mkdir(nestedDir, 0700)
	nestedTestFiles := make([]string, 10)
	for i := 0; i < 10; i++ {
		testFileName := fmt.Sprintf("test_%d.md", i)
		nestedTestFiles[i] = testFileName
		os.Create(filepath.Join(nestedDir, testFileName))
	}

	walker = &indexWalker{}
	filepath.Walk(testPath, walker.walk)

	if len(walker.dirToFiles) != 2 {
		t.Logf("Expected only one entry, got %d\n", len(walker.dirToFiles))
		t.Fail()
	}
	expected = map[string][]string{
		testPath:  testFiles,
		nestedDir: nestedTestFiles,
	}

	if !reflect.DeepEqual(expected, walker.dirToFiles) {
		t.Logf("Expected map to equal '%+#v', got: '%+#v'\n", expected, walker.dirToFiles)
		t.Fail()
	}
}

type cleanup func()

type testData struct {
	root     string
	path     string
	filename string
	count    int
	content  string
}

type testDataSet []testData

func (t testDataSet) getTestDataForPath(path string) testDataSet {
	filteredSet := make(testDataSet, 0)
	for _, v := range t {
		if v.path == path {
			filteredSet = append(filteredSet, v)
		}
	}
	return filteredSet
}

var defaultContent string = `
	# Header

	Text
	`
var defaultFilename string = "test.md"

func createTestDataFor(root string, paths ...string) cleanup {
	os.Mkdir(filepath.Join("/tmp", root), 0700)
	for _, p := range paths {
		testData := testData{
			root:     root,
			path:     p,
			content:  defaultContent,
			filename: defaultFilename,
		}
		setupTestData(testData)
	}
	return func() {
		os.RemoveAll(root)
	}
}

func setupTestData(data testData) {
	path := data.path
	if path != "" {
		os.Mkdir(filepath.Join(data.root, path), 0700)
	} else {
		path = data.root
	}
	count := data.count
	if count <= 0 {
		count = 1
	}
	for i := 0; i < data.count; i++ {
		fileName := fmt.Sprintf("%s_%d.md", data.filename, i)
		testFilePath := filepath.Join(path, fileName)
		testFile, _ := os.Create(testFilePath)
		testFile.WriteString(data.content)
		testFile.Close()
	}
}

func newTestServer(handler http.Handler) (addr string, c cleanup) {
	server := httptest.NewServer(handler)
	return server.URL, func() { server.Close() }
}
