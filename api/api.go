package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

var CF_KEY = os.Getenv("CURSEFORGE_KEY")
var client = &http.Client{Transport: &cfTransport{}}

const MINECRAFT_ID = 432

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
		errors.New(fmt.Sprintf("\nREQUEST:\n%s\nRESPONSE:\n%s\n----------------\n", string(req), string(res))),
	)
	return errors.Join(errs...)
}

func dumpJson(body []byte, errs ...error) error {
	var pretty bytes.Buffer
	err := json.Indent(&pretty, body, "", "  ")
	errs = append(
		errs,
		err,
		errors.New(fmt.Sprintf("\nJSON:\n%s\n----------------\n", string(pretty.Bytes()))),
	)
	return errors.Join(errs...)
}

func getJSON[T any](ret *T, url string) error {
	res, err := client.Get(url)
	if err != nil {
		return dumpHttp(res, err)
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

const nextMajor int = 20

var memVersions = []string{
	"1.19.4",
	"1.19.3",
	"1.19.2",
	"1.19.1",
	"1.19",
	"1.18.2",
	"1.18.1",
	"1.18",
	"1.17.1",
	"1.17",
	"1.16.5",
	"1.16.4",
	"1.16.3",
	"1.16.2",
	"1.16.1",
	"1.16",
	"1.15.2",
	"1.15.1",
	"1.15",
	"1.14.4",
	"1.14.3",
	"1.14.2",
	"1.14.1",
	"1.14",
	"1.13.2",
	"1.13.1",
	"1.13",
	"1.12.2",
	"1.12.1",
	"1.12",
	"1.11.2",
	"1.11.1",
	"1.11",
	"1.10.2",
	"1.10.1",
	"1.10",
	"1.9.4",
	"1.9.3",
	"1.9.2",
	"1.9.1",
	"1.9",
	"1.8.9",
	"1.8.8",
	"1.8.7",
	"1.8.6",
	"1.8.5",
	"1.8.4",
	"1.8.3",
	"1.8.2",
	"1.8.1",
	"1.8",
	"1.7.10",
	"1.7.9",
	"1.7.8",
	"1.7.7",
	"1.7.6",
	"1.7.5",
	"1.7.4",
	"1.7.3",
	"1.7.2",
	"1.6.4",
	"1.6.2",
	"1.6.1",
	"1.5.2",
	"1.5.1",
	"1.4.7",
	"1.4.6",
	"1.4.5",
	"1.4.4",
	"1.4.2",
	"1.3.2",
	"1.3.1",
	"1.2.5",
	"1.2.4",
	"1.2.3",
	"1.2.2",
	"1.2.1",
	"1.1",
	"1.0",
}

var modLoaderKeywords = []string{
	"Any",
	"Forge",
	"Cauldron",
	"LiteLoader",
	"Fabric",
	"Quilt",
	"NeoForge",
	"InvalidModLoader",
}

type FileRelation int

const (
	EmbeddedLibrary FileRelation = iota + 1
	OptionalDependency
	RequiredDependency
	Tool
	Incompatible
	Include
)

type ReleaseType int

const (
	Release ReleaseType = iota + 1
	Beta
	Alpha
)

type Dependency struct {
	ModId    int          `json:"modId"`
	Relation FileRelation `json:"relationType"`
}

type ModFiles struct {
	ID          int
	ModLoader   int
	GameVersion string
	Files       []CfFile
}

type gameVersion struct {
	Version string `json:"versionString"`
}

type CfResponse[D any] struct {
	Data D `json:"data"`
}

type CfFile struct {
	Uploaded          time.Time    `json:"fileDate"`
	ID                int          `json:"id"`
	Name              string       `json:"fileName"`
	Size              int          `json:"fileSizeOnDisk"`
	DownloadURL       string       `json:"downloadUrl"`
	SupportedVersions []string     `json:"gameVersions"`
	Dependencies      []Dependency `json:"dependencies"`
	Release           ReleaseType  `json:"releaseType"`
}

type cfMod struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Summary       string    `json:"summary"`
	DownloadCount int       `json:"downloadCount"`
	Likes         int       `json:"thumbsUpCount"`
	Rating        int       `json:"rating"`
	Created       time.Time `json:"dateCreated"`
	Modified      time.Time `json:"dateModified"`
	Released      time.Time `json:"dateReleased"`
	Files         []CfFile  `json:"latestFiles"`
}

type cfGameVersion struct {
	ID                    int       `json:"id"`
	GameVersionID         int       `json:"gameVersionId"`
	VersionString         string    `json:"versionString"`
	JarDownloadURL        string    `json:"jarDownloadUrl"`
	JSONDownloadURL       string    `json:"jsonDownloadUrl"`
	Approved              bool      `json:"approved"`
	DateModified          time.Time `json:"dateModified"`
	GameVersionTypeID     int       `json:"gameVersionTypeId"`
	GameVersionStatus     int       `json:"gameVersionStatus"`
	GameVersionTypeStatus int       `json:"gameVersionTypeStatus"`
}
