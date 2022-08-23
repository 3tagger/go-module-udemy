package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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
