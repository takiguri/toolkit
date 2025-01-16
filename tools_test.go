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
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("Wrong length random string returned")
	}
}

var uploadTests = []struct {
	name             string
	allowedTypes     []string
	shouldRenameFile bool
	isErrorExpected  bool
}{
	{
		name:             "allowed no rename",
		allowedTypes:     []string{"image/jpeg", "image/png"},
		shouldRenameFile: false,
		isErrorExpected:  false,
	},
	{
		name:             "allowed rename",
		allowedTypes:     []string{"image/jpeg", "image/png"},
		shouldRenameFile: true,
		isErrorExpected:  false,
	},
	{
		name:             "not allowed",
		allowedTypes:     []string{"image/jpeg"},
		shouldRenameFile: false,
		isErrorExpected:  true,
	},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// Create the form data field 'file'
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
				t.Error(err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which receives data
		r := httptest.NewRequest("POST", "/", pr)
		r.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(r, "./testdata/uploads/", e.shouldRenameFile)
		if err != nil && !e.isErrorExpected {
			t.Error(err)
		}

		if !e.isErrorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFilename)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			// clean-up
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFilename))
		}

		if !e.isErrorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// Create the form data field 'file'
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
			t.Error(err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read from the pipe which receives data
	r := httptest.NewRequest("POST", "/", pr)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools
	testTools.AllowedFileTypes = []string{"image/png"}

	f, err := testTools.UploadOneFile(r, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", f.NewFilename)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	// clean-up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", f.NewFilename))
}
