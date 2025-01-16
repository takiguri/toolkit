package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module.
// Any variable of this type will have access to all the methods with the receiver *Tool
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

// RandomString returns a string of random characters of length n, using randomStringSource
// as the source for the string
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)
}

// Struct that holds information about an uploaded file
type UploadedFile struct {
	NewFilename      string
	OriginalFilename string
	FileSize         int64
}

// Uploads a single file from a multipart-form data POST request
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, shouldRenameFile bool) (*UploadedFile, error) {
	files, err := t.UploadFiles(r, uploadDir, shouldRenameFile)
	if err != nil {
		return nil, err
	}
	return files[0], nil
}

// Uploads files from a multipart-form data POST request
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, shouldRenameFile bool) ([]*UploadedFile, error) {
	// 1: Parse the request body as multipart-form data
	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("file size is too large")
	}

	// 2: Iterate through all files
	var uploadedFiles []*UploadedFile
	for _, fhdr := range r.MultipartForm.File {
		for _, hdr := range fhdr {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				// 3: Validate mime types
				infile, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				buff := make([]byte, 512)
				_, err = infile.Read(buff)
				if err != nil {
					return nil, err
				}

				filetype := http.DetectContentType(buff)
				isFileAllowed := false

				if len(t.AllowedFileTypes) > 0 {
					for _, ft := range t.AllowedFileTypes {
						if strings.EqualFold(filetype, ft) {
							isFileAllowed = true
						}
					}
				}

				if !isFileAllowed {
					return nil, errors.New("uploading filetype is not permitted")
				}

				// 4: Rename when necessary
				var uploadedFile UploadedFile
				uploadedFile.OriginalFilename = hdr.Filename

				if shouldRenameFile {
					uploadedFile.NewFilename = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFilename = hdr.Filename
				}

				// 5: Write to disk
				var outfile *os.File
				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFilename)); err != nil {
					return nil, err
				} else {
					filesize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = filesize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)
				return uploadedFiles, nil
			}(uploadedFiles)
			// if an error occurs in the middle of the iteration,
			// return the premature list and the error
			if err != nil {
				return uploadedFiles, err
			}
		}
	}

	// If all goes well, return the slice
	return uploadedFiles, nil
}
