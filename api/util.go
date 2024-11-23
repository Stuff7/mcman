package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stuff7/mcman/slc"
	"github.com/stuff7/mcman/storage"
)

var CF_KEY = os.Getenv("CURSEFORGE_KEY")
var client = &http.Client{Transport: &cfTransport{}}

const MINECRAFT_ID = 432

func unzip(src, dest string) error {
	r, err := storage.OpenZip(src)
	if err != nil {
		return fmt.Errorf("failed to unzip from %#+v to %#+v: %w", src, dest, err)
	}
	defer r.Close()

	if err := storage.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for _, file := range r.File {
		filePath := filepath.Join(dest, file.Name)

		if file.FileInfo().IsDir() {
			if err := storage.MkdirAll(filePath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			if err := extractFile(file, filePath); err != nil {
				return err
			}
		}
	}

	return nil
}

func extractFile(file *zip.File, filePath string) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry: %w", err)
	}
	defer src.Close()

	if err := storage.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	dest, err := storage.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dest.Close()

	progressReader := &ProgressReader{
		Reader: src,
		Ui: ProgressBar{
			Description: "Extracting",
			Name:        file.Name,
			Total:       file.FileInfo().Size(),
			Progress:    0,
		},
	}

	if _, err := io.Copy(dest, progressReader); err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	return nil
}

func downloadFile(url string, name string, desc ...string) (bool, error) {
	if _, err := storage.Stat(name); err == nil {
		return false, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer res.Body.Close()

	contentLength := res.Header.Get("Content-Length")
	totalSize := int64(-1)
	if contentLength != "" {
		totalSize, err = strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			totalSize = -1
		}
	}

	file, err := storage.Create(name)
	if err != nil {
		return true, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	progressReader := &ProgressReader{
		Reader: res.Body,
		Ui: ProgressBar{
			Description: "Downloading",
			Name:        name,
			Total:       totalSize,
			Progress:    0,
		},
	}

	if d := slc.Get(desc, 0); d != nil {
		progressReader.Ui.Description = *d
	}

	_, err = io.Copy(file, progressReader)
	if err != nil {
		return true, fmt.Errorf("failed to copy data: %w", err)
	}

	return true, nil
}

type cfTransport struct{}

func (t *cfTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "https"
	req.URL.Host = "api.curseforge.com"
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", CF_KEY)
	return http.DefaultTransport.RoundTrip(req)
}

func dumpHttp(r *http.Response, errs ...error) error {
	req, err := httputil.DumpRequest(r.Request, true)
	res, err := httputil.DumpResponse(r, true)
	errs = append(
		errs,
		err,
		fmt.Errorf("\nREQUEST:\n%s\nRESPONSE:\n%s\n----------------\n", string(req), string(res)),
	)
	return errors.Join(errs...)
}

func dumpJson(body []byte, errs ...error) error {
	var pretty bytes.Buffer

	err := json.Indent(&pretty, body, "", "  ")
	jsonErr := fmt.Errorf("\nJSON:\n%s\n----------------\n", string(pretty.Bytes()))
	if len(errs) == 0 {
		return jsonErr
	}

	errs = append(
		errs,
		err,
		jsonErr,
	)

	return errors.Join(errs...)
}

func getJSON[T any](ret *T, url string) error {
	res, err := client.Get(url)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return dumpHttp(res, errors.New("Bad Response"))
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return dumpHttp(res, err)
	}

	var apiRes CfResponse[T]
	if err := json.Unmarshal(body, &apiRes); err != nil {
		return dumpHttp(res, dumpJson(body, err))
	}

	*ret = apiRes.Data
	return nil
}

type ProgressBar struct {
	Description string
	Name        string
	Total       int64
	Progress    int64
}

func (pr *ProgressBar) printProgress() {
	if pr.Total > 0 {
		percent := float64(pr.Progress) / float64(pr.Total) * 100
		fmt.Printf("\x1b[2K%s %3.2f%% %s%#+v%s\r", pr.Description, percent, BOLD, pr.Name, RESET)
	} else {
		fmt.Printf("\x1b[2K%s %d bytes\r", pr.Description, pr.Progress)
	}
}

type ProgressReader struct {
	Reader io.Reader
	Ui     ProgressBar
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Ui.Progress += int64(n)
	pr.Ui.printProgress()
	return n, err
}

func pluralize(w string, n int) string {
	if n == 1 {
		return w
	} else {
		return w + "s"
	}
}
