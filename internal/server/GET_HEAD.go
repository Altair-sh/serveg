package server

import (
	"fmt"
	"io"
	"os"
	"path"
	"serveg/internal/html"
	"serveg/internal/http"
)

func (conn *TcpConnection) GET_HEAD(req *http.Request) *http.Response {
	path, share, urlStatus := conn.srv.Conf.Share.MapUrl(req.Url)
	// handle authorization
	if share != nil {
		access := checkSharePasswords(share, req)
		if access < AccessReadOnly {
			return conn.createErrorResponse(req, false,
				http.StatusUnauthorized, "reader password required").
				WithHeader("www-authenticate",
					`Basic realm="`+share.Url+` (read)", charset="UTF-8"`)
		}
	}

	switch urlStatus {
	case UrlFile:
		return conn.serveFile(req, path)
	case UrlDir:
		// show upload form if the share accepts uploads
		writable := share != nil && share.WritePassword != ""
		return conn.serveDirListing(req, path, writable)
	case UrlShareNotFound, UrlDoesntExistInShare:
		return conn.createErrorResponse(req, false, http.StatusNotFound, req.Url)
	case UrlForbidden:
		return conn.createErrorResponse(req, false, http.StatusForbidden, req.Url)
	case UrlIncorrect:
		return conn.createErrorResponse(req, false, http.StatusBadRequest,
			"invalid url '"+req.Url+"'")
	default:
		panic(fmt.Sprintf("unsupported UrlStatus: %d", int32(urlStatus)))
	}
}

func (conn *TcpConnection) serveDirListing(req *http.Request, dirPath string, writable bool) *http.Response {
	htmlBytes, err := html.GenDirListingHtml(dirPath,
		req.Url, conn.srv.Conf.HtmlTitle, writable)
	if err != nil {
		return conn.createErrorResponse(req, false, http.StatusInternalServerError, err.Error())
	}

	return conn.http.CreateResponse(http.StatusOK, req, false).
		WithBodyBuf(htmlBytes, http.TextHtml)
}

func (conn *TcpConnection) serveFile(req *http.Request, filePath string) *http.Response {
	f, err := os.Open(filePath)
	if err != nil {
		return conn.createErrorResponse(req, false, http.StatusInternalServerError, err.Error())
	}
	var fileOwnershipTransferred bool
	defer func() {
		if !fileOwnershipTransferred {
			_ = f.Close()
		}
	}()

	finf, err := f.Stat()
	if err != nil {
		return conn.createErrorResponse(req, false, http.StatusInternalServerError, err.Error())
	}
	size := finf.Size()

	// Clients send back the exact last-modified value they received.
	// Exact match (not "<=") also detects a file replaced by one with older mtime.
	lastModified := finf.ModTime().UTC().Format(http.HeaderTimeFormat)
	if req.Headers["if-modified-since"] == lastModified {
		return conn.http.CreateResponse(http.StatusNotModified, req, false)
	}

	// Range is used to continue interrupted downloads.
	// If-Range says "send the range only if the file wasn't changed, otherwise send the whole file".
	rangeHdr := req.Headers["range"]
	if ifRange, ok := req.Headers["if-range"]; ok && ifRange != lastModified {
		rangeHdr = ""
	}
	start, end, rangeResult := http.ParseRange(rangeHdr, size)

	status := http.StatusOK
	bodyLen := size
	switch rangeResult {
	case http.RangeUnsatisfiable:
		return conn.createErrorResponse(req, false, http.StatusRequestedRangeNotSatisfiable,
			fmt.Sprintf("range '%s' is outside of file size %d", rangeHdr, size)).
			WithHeader("content-range", fmt.Sprintf("bytes */%d", size)).
			WithHeader("accept-ranges", "bytes")
	case http.RangeOK:
		status = http.StatusPartialContent
		bodyLen = end - start + 1
		if _, err = f.Seek(start, io.SeekStart); err != nil {
			return conn.createErrorResponse(req, false, http.StatusInternalServerError, err.Error())
		}
	}

	res := conn.http.CreateResponse(status, req, false)
	fileName := path.Base(filePath)
	mimeType := http.GetMimeType(fileName)

	if req.Method != "HEAD" {
		// file will be closed by SendResponse()
		fileOwnershipTransferred = true
		res.WithBodyReader(f, bodyLen, mimeType)
	} else {
		// WithBodyReader does nothing for HEAD, but clients use HEAD to get file size
		res.WithHeaderI("content-length", bodyLen).
			WithHeader("content-type", mimeType)
	}
	if rangeResult == http.RangeOK {
		res.Headers["content-range"] = fmt.Sprintf("bytes %d-%d/%d", start, end, size)
	}
	// Tell browser to not use its own caching, remember last-modified
	// On next request browser will send if-modified-since and let server check if file changed.
	res.Headers["cache-control"] = "no-cache"
	res.Headers["last-modified"] = lastModified
	res.Headers["content-disposition"] = http.ContentDisposition("inline", fileName)
	res.Headers["accept-ranges"] = "bytes"

	return res
}
