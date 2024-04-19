package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/stuff7/mcman/slc"
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

func downloadFile(url string, name string) (bool, error) {
	if _, err := os.Stat(name); err == nil {
		return false, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return false, dumpHttp(res, err)
	}
	defer res.Body.Close()

	file, err := os.Create(name)
	if err != nil {
		return true, err
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return true, err
	}

	return true, nil
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

type modEntry struct {
	Id          int       `json:"id"`
	ModLoader   int       `json:"modLoader"`
	GameVersion string    `json:"gameVersion"`
	Name        string    `json:"name"`
	DownloadUrl string    `json:"downloadUrl"`
	Deps        []int     `json:"deps"`
	Uploaded    time.Time `json:"uploaded"`
}

func appendModEntry(mods []modEntry, id int, query searchQuery, f *CfFile) []modEntry {
	if !slices.ContainsFunc(mods, func(m modEntry) bool { return id == m.Id }) {
		return append(mods, modEntry{
			Id:          id,
			ModLoader:   query.ModLoader,
			GameVersion: query.GameVersion,
			Name:        f.Name,
			DownloadUrl: tryGetURL(f),
			Uploaded:    f.Uploaded,
			Deps: slc.Map(
				slc.Filter(f.Dependencies, func(d Dependency) bool { return d.Relation == RequiredDependency }),
				func(d Dependency) int { return d.ModId },
			),
		})
	}
	return mods
}

func tryGetURL(f *CfFile) string {
	if f.DownloadURL == nil {
		fmt.Printf("%s! %sMissing Download URL for mod %+v. Trying to guess it%s\n", clr(227), BOLD, f.Name, RESET)
		var ids [2]int
		if f.ID > 999999 {
			ids[0] = f.ID / 1000
			ids[1] = f.ID % 1000
		} else if f.ID > 99999 {
			ids[0] = f.ID / 100
			ids[1] = f.ID % 100
		} else {
			ids[0] = f.ID / 100
			ids[1] = f.ID % 10
		}

		return fmt.Sprintf("%s%d/%03d/%s", downloadURL, ids[0], ids[1], url.QueryEscape(f.Name))
	}

	return *f.DownloadURL
}

func listMods(mods []modEntry, filter func(m modEntry) bool) {
	var count int
	var sb strings.Builder
	for i, m := range mods {
		if filter == nil || filter(m) {
			count++
			sb.WriteString(fmt.Sprintf("\n%s%03d%s %s%s%s # %s%d%s", clr(157)+BOLD, i, RESET, clr(214)+BOLD, m.Name, RESET, clr(157), m.Id, RESET))
			sb.WriteString(fmt.Sprintf(" [%s%s %s%s%s]\n", clr(228)+BOLD, modLoaderKeywords[m.ModLoader], clr(231), m.GameVersion, RESET))
			if len(m.Deps) > 0 {
				deps := slc.Map(slc.Filter(mods, func(d modEntry) bool {
					return slices.Contains(m.Deps, d.Id)
				}), func(d modEntry) string {
					return fmt.Sprintf("%s%d%s#%s%s%s", clr(213)+BOLD, d.Id, clr(219), clr(157), d.Name, RESET)
				})
				sb.WriteString(fmt.Sprintf("Deps:     %v\n", deps))
			}
			sb.WriteString(fmt.Sprintf("Download: %s%s%s\n", clr(123)+BOLD, m.DownloadUrl, RESET))
			sb.WriteString(fmt.Sprintf("Uploaded: %s%s%s\n", clr(219)+BOLD, m.Uploaded.Format(time.RFC822), RESET))
		}
	}

	fmt.Printf("Found %s%d%s mods\n", clr(49), count, RESET)
	println(sb.String())
}

func collectDeps(mods []modEntry, id int, rem *[]int) error {
	idx := slices.IndexFunc(mods, func(m modEntry) bool { return m.Id == id })
	if idx == -1 {
		return nil
	}
	mod := &mods[idx]
	if len(mod.Deps) > 0 {
		*rem = append(*rem, mod.Deps...)
		for _, d := range mod.Deps {
			collectDeps(mods, d, rem)
		}
	}

	return nil
}

func removeModEntry(mods *[]modEntry, idx int) error {
	if idx < 0 || idx >= len(*mods) {
		return fmt.Errorf("Not found")
	}

	mod := &(*mods)[idx]
	rem := append([]int{mod.Id}, mod.Deps...)
	for i := 1; i < len(rem); i++ {
		collectDeps(*mods, rem[i], &rem)
	}

	for i := 0; i < len(*mods); i++ {
		m := &(*mods)[i]
		if mod.Id == m.Id {
			continue
		}
		for i, r := range rem {
			if slices.Contains(rem, m.Id) || !slices.Contains(m.Deps, r) {
				continue
			}
			if mod.Id == r {
				return fmt.Errorf("Cannot remove mod %#+v because other mods depend on it", mod.Name)
			}
			rem = slices.Delete(rem, i, i+1)
		}
	}

	*mods = slc.Filter(*mods, func(m modEntry) bool {
		remove := slices.Contains(rem, m.Id)
		if remove {
			if m.Id != mod.Id {
				fmt.Printf("%s- Dep %s%s%s removed\n", clr(216), BOLD, m.Name, RESET)
			} else {
				fmt.Printf("%s- Mod %s%s%s removed\n", clr(219), BOLD, m.Name, RESET)
			}
		}
		return !remove
	})

	return nil
}

type ModFiles struct {
	ID          int
	ModLoader   int
	GameVersion string
	Files       []CfFile
}

type importFile struct {
	ID int `json:"id"`
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
	Size              int          `json:"fileLength"`
	DownloadURL       *string      `json:"downloadUrl"`
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
