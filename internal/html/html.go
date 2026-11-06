package html

import (
	"bytes"
	_ "embed"
	"html/template"
	"os"
	"path"
	"serveg/internal/disk_usage"
	"serveg/internal/http"
	sh "serveg/internal/string_helper"
)

const DateTimeTZ = "2006-01-02 15:04:05 UTC-07"

// The template file is embedded into the binary at build time.
//
//go:embed ListingPage.gohtml
var listingPageSrc string

// Parsed once at startup, Execute() is safe for concurrent use.
// Must() panics on syntax error, so a broken template fails at start, not on request.
var listingPageTemplate = template.Must(template.New("ListingPage").Parse(listingPageSrc))

// Data for ListingPage.gohtml. Fields must be exported to be visible in the template.
// Values are not escaped: html/template does it.
type listingPage struct {
	Title     string
	Path      string // directory url, ends with '/'
	PathUrl   string // percent-encoded Path
	ParentUrl string // percent-encoded, ends with '/'
	Writable  bool   // show the upload form
	FreeSpace string // formatted with HumanBytes
	Entries   []listingEntry
}

type listingEntry struct {
	Name  string
	Url   string // percent-encoded, without trailing '/'
	Size  string // formatted with HumanBytes, empty for directories
	Date  string // formatted with DateTimeTZ
	IsDir bool
}

// reqUrl is a clean url without trailing slash (except "/"), see sh.ResolveSafePath
func GenDirListingHtml(dirPath, reqUrl, pageTitle string, writable bool) ([]byte, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	dirUrl := withTrailingSlash(reqUrl)
	page := listingPage{
		Title:     pageTitle + " " + dirUrl,
		Path:      dirUrl,
		PathUrl:   http.PercentEcodeUrl(dirUrl),
		ParentUrl: http.PercentEcodeUrl(withTrailingSlash(path.Dir(reqUrl))),
		Writable:  writable,
		Entries:   make([]listingEntry, 0, len(dirEntries)),
	}

	// listing is still useful without free space, so error is shown on the page
	if usage, err := disk_usage.DiskUsage(dirPath); err == nil {
		page.FreeSpace = sh.HumanBytes(usage.Free)
	} else {
		page.FreeSpace = "unknown (" + err.Error() + ")"
	}

	// directories first, then files, both sorted by name (os.ReadDir sorts)
	var files []listingEntry
	for _, e := range dirEntries {
		einf, err := e.Info()
		if err != nil {
			return nil, err
		}
		entry := listingEntry{
			Name:  e.Name(),
			Url:   http.PercentEcodeUrl(dirUrl + e.Name()),
			Date:  einf.ModTime().Format(DateTimeTZ),
			IsDir: e.IsDir(),
		}
		if entry.IsDir {
			page.Entries = append(page.Entries, entry)
		} else {
			entry.Size = sh.HumanBytes(einf.Size())
			files = append(files, entry)
		}
	}
	page.Entries = append(page.Entries, files...)

	// render to buffer, not to socket: content-length must be known,
	// and an error in the middle must not send a half page
	var buf bytes.Buffer
	buf.Grow(len(listingPageSrc) + len(page.Entries)*200)
	if err := listingPageTemplate.Execute(&buf, &page); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func withTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s
	}
	return s + "/"
}
