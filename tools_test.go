package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var tools Tools

	testcase := []struct {
		name           string
		input          int
		expectedLength int
	}{
		{
			name:           "zero length",
			input:          0,
			expectedLength: 0,
		},
		{
			name:           "100 length",
			input:          100,
			expectedLength: 100,
		},
	}

	for _, tc := range testcase {
		got := tools.RandomString(tc.input)

		if len(got) != tc.expectedLength {
			t.Errorf("expecting RandomString to return a string with length of %d, got %d", tc.expectedLength, len(got))
		}
	}
}

func TestTools_UploadFiles(t *testing.T) {
	var uploadTests = []struct {
		name          string
		allowedTypes  []string
		renameFile    bool
		errorExpected bool
	}{
		{
			name:          "allowed no rename",
			allowedTypes:  []string{"image/jpeg", "image/png"},
			renameFile:    false,
			errorExpected: false,
		},
		{
			name:          "allowed rename",
			allowedTypes:  []string{"image/jpeg", "image/png"},
			renameFile:    true,
			errorExpected: false,
		},
		{
			name:          "not allowed",
			allowedTypes:  []string{},
			renameFile:    false,
			errorExpected: true,
		},
	}

	for _, e := range uploadTests {
		// set up a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			// simulating uploading png file
			defer writer.Close()
			defer wg.Done()

			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		testTools := Tools{
			AllowedFileTypes: e.allowedTypes,
		}

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if err != nil {
				t.Errorf("%s: error not expected but received", e.name)
			} else {
				if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
					t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
				}

				if e.renameFile {
					if uploadedFiles[0].NewFileName == "img.png" {
						t.Errorf("%s: expected file to have name changed, got %q", e.name, uploadedFiles[0].NewFileName)
					}
				} else {
					if uploadedFiles[0].NewFileName != "img.png" {
						t.Errorf("%s: expected file to have name not changed, got %q", e.name, uploadedFiles[0].NewFileName)
					}
				}

				// clean up
				_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
			}
		}
		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		// simulating uploading png file
		defer writer.Close()

		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	testTools := Tools{
		AllowedFileTypes: []string{"image/png"},
	}

	file, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", file.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	if file.NewFileName == "img.png" {
		t.Errorf("expected file to have name changed, got %q", file.NewFileName)
	}

	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", file.NewFileName))

}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools

	dir := "./testdata/myDir"

	err := testTool.CreateDirIfNotExist(dir)
	if err != nil {
		t.Error(err.Error())
	}

	err = testTool.CreateDirIfNotExist(dir)
	if err != nil {
		t.Error(err.Error())
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("directory %q expected to be exist, but it doesn't", dir)
	}

	os.Remove(dir)
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	testCases := []struct {
		name     string
		input    string
		expected string
		hasErr   bool
	}{
		{
			name:     "empty string, error",
			input:    "",
			expected: "",
			hasErr:   true,
		},
		{
			name:     "standard string, no error",
			input:    "this should be slug 123",
			expected: "this-should-be-slug-123",
			hasErr:   false,
		},
		{
			name:     "all non alphanumeric characters, error",
			input:    "!@*(!*@$(!&(@!&<><>><",
			expected: "",
			hasErr:   true,
		},
		{
			name:     "japanese string, error",
			input:    "こんにちは世界",
			expected: "",
			hasErr:   true,
		},
		{
			name:     "standard and japanese string, no error",
			input:    "hello world こんにちは世界",
			expected: "hello-world",
			hasErr:   false,
		},
	}

	for _, tc := range testCases {
		got, err := testTool.Slugify(tc.input)
		if got != tc.expected {
			t.Errorf("%s: expected %q, got %q", tc.name, tc.expected, got)
		}
		if tc.hasErr && err == nil {
			t.Errorf("%s: expecting got an error, didn't get any", tc.name)
		} else if !tc.hasErr && err != nil {
			t.Errorf("%s: expecting no error, got one: %q", tc.name, err)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools

	testTool.DownloadStaticFile(rr, req, "./testdata", "pic.jpg", "puppy.jpg")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.jpg\"" {
		t.Error("wrong content disposition of", res.Header["Content-Disposition"][0])
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

func TestTools_ReadJSON(t *testing.T) {
	testcases := []struct {
		name          string
		json          string
		errorExpected bool
		maxSize       int64
		allowUnknown  bool
	}{
		{
			name:          "good json",
			json:          `{"foo":"bar"}`,
			errorExpected: false,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "badly formatted json",
			json:          `{"foo":}`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "incorrect type json",
			json:          `{"foo": 1}`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "double json",
			json:          `{"foo": "bar"}{"foo": "bar"}`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "empty json",
			json:          ``,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "syntax error in json",
			json:          `{"foo": 1"`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "syntax error in json",
			json:          `{"foo": 1"`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "syntax error in json",
			json:          `{"bar": "bar"}`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "allow unknown field in json",
			json:          `{"bar": "bar"}`,
			errorExpected: false,
			maxSize:       1024,
			allowUnknown:  true,
		},
		{
			name:          "size too large json",
			json:          `{"foo": "bar"}`,
			errorExpected: true,
			maxSize:       1,
			allowUnknown:  false,
		},
		{
			name:          "missing field name json",
			json:          `{foo: "bar"}`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
		{
			name:          "not json",
			json:          `hello there`,
			errorExpected: true,
			maxSize:       1024,
			allowUnknown:  false,
		},
	}

	testTool := Tools{}

	for _, tc := range testcases {
		testTool.MaxJSONSize = tc.maxSize
		testTool.AllowUnknownFields = tc.allowUnknown

		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(tc.json)))
		if err != nil {
			t.Log("error: ", err)
		}

		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)
		if err != nil && !tc.errorExpected {
			t.Errorf("%s: expecting no error, got error: %s", tc.name, err)
		}

		if err == nil && tc.errorExpected {
			t.Errorf("%s: expecting error, got no error", tc.name)
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonData := JSONResponse{
		Error:   false,
		Message: "Foo",
		Data:    "Bar",
	}

	toolTest := Tools{}

	header := http.Header{}
	header.Set("FOO", "BAR")

	err := toolTest.WriteJSON(rr, http.StatusOK, jsonData, header)
	if err != nil {
		t.Error("not expecting any error, got: ", err)
	}

	if rr.Result().Header.Get("FOO") != "BAR" {
		t.Errorf("expecting header to have %q, there is none", "BAR")
	}

	if rr.Result().Header.Get("Content-Type") == "application/json" {
		t.Errorf("expecting header to have %q, there is none", "application/json")
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	testTools := Tools{}

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("test error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	err = json.NewDecoder(rr.Body).Decode(&payload)
	if err != nil {
		t.Error(err)
	}

	if !payload.Error {
		t.Errorf("expected to have error")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected to have error %q, got %q", http.StatusServiceUnavailable, rr.Code)
	}
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("ok")),
			Header:     http.Header{},
		}
	})

	testTools := Tools{}
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"

	_, err := testTools.PushJSONToRemote("http://example.some.path", foo, client)
	if err != nil {
		t.Error("There should be no error: ", err)
	}
}
