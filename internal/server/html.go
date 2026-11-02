package server

import (
	"os"
	"path"
	"serveg/internal/http"
	"serveg/internal/log"
	sh "serveg/internal/string_helper"
)

const DateTimeTZ = "2006-01-02 15:04:05 UTC-07"

func htmlEscape(s string) string {
	sb := sh.StringBuilder{Buf: make([]byte, 0, len(s))}
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch b {
		case '&':
			sb.AddS("&amp;")
		case '<':
			sb.AddS("&lt;")
		case '>':
			sb.AddS("&gt;")
		case '"':
			sb.AddS("&quot;")
		case '\'':
			sb.AddS("&#39;")
		default:
			sb.AddB(b)
		}
	}
	return sb.StringUnsafe()
}

type listingRow struct {
	name  string // html-escaped
	url   string // url-encoded
	mtime string // formatted with DateTimeTZ
	size  string // formatted with HumanBytes
	isDir bool
}

func (conn *connection) serveDirListing(dirPath string) *http.Response {
	htmlBytes, err := genDirListingHtml(&conn.log, dirPath, conn.req.Url)
	if err != nil {
		return conn.createErrorResponse(false, http.StatusInternalServerError, err.Error())
	}

	res := conn.http.CreateResponse(http.StatusOK, conn.req, false)
	res.AttachBodyBuf(htmlBytes, http.TextHtml)
	return res
}

func genDirListingHtml(logger *log.LoggerContext, dirPath, reqUrl string) ([]byte, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var dirs, files []listingRow
	var estimatedLen = 460 // html for page without entries
	for _, e := range dirEntries {
		einf, err := e.Info()
		if err != nil {
			return nil, err
		}

		row := listingRow{
			name:  htmlEscape(e.Name()),
			url:   http.PercentEcodeUrl(path.Join(reqUrl, e.Name())),
			mtime: einf.ModTime().Format(DateTimeTZ),
			isDir: e.IsDir(),
		}
		if row.isDir {
			row.size = "DIR"
			dirs = append(dirs, row)
		} else {
			row.size = sh.HumanBytes(einf.Size())
			files = append(files, row)
		}

		estimatedLen += len(row.name) + len(row.url) + len(row.mtime) + len(row.size)
		estimatedLen += 100 // html for row
	}

	sb := sh.StringBuilder{Buf: make([]byte, 0, estimatedLen)}
	sb.AddS(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>`)
	reqUrlHtmlEscaped := htmlEscape(reqUrl)
	sb.AddS(reqUrlHtmlEscaped)
	sb.AddS(`</title>
<style>
	body { font-family: sans-serif; margin: 2em; }
	td { padding: .2em 1em .2em 0; }
	td.size, td.date { color: #666; white-space: nowrap; }
</style>
</head>
<body>`)
	sb.AddS("<h1>").AddS(reqUrlHtmlEscaped).AddS("</h1>")
	sb.AddS(`<table id="listing">`)
	sb.AddS(`<tr><td><a href="`)
	sb.AddS(http.PercentEcodeUrl(path.Dir(reqUrl)))
	sb.AddS(`">../</a></td></tr>`)

	for i := 0; i < len(dirs); i++ {
		genListingRowHtml(&sb, &dirs[i])
	}
	for i := 0; i < len(files); i++ {
		genListingRowHtml(&sb, &files[i])
	}

	sb.AddS(`
</table>
</body>
</html>`)

	realLen := sb.Len()
	if estimatedLen < realLen {
		logger.Warn("genDirListingHtml(): size prediction failed. estimatedLen: ",
			estimatedLen, " realLen: ", realLen)
	}
	return sb.Buf, nil
}

func genListingRowHtml(sb *sh.StringBuilder, row *listingRow) {
	sb.AddS(`<tr>`)
	sb.AddS(`<td><a href="`).AddS(row.url).AddB('"')
	if !row.isDir {
		// tell browser to download file instead of opening in new tab
		sb.AddS(" download")
	}
	sb.AddB('>').AddS(row.name).AddS(`</a></td>`)
	sb.AddS(`<td class="size">`).AddS(row.size).AddS(`</td>`)
	sb.AddS(`<td class="date">`).AddS(row.mtime).AddS(`</td>`)
	sb.AddS(`</tr>`)
}
