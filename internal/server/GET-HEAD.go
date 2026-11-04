package server

import (
	"fmt"
	"os"
	"path"
	"serveg/internal/html"
	"serveg/internal/http"
	"time"
)

func (conn *TcpConnection) serveFileOrDir(req *http.Request) *http.Response {
	path, share, urlStatus := conn.srv.Conf.Share.MapUrl(req.Url)
	// TODO implement authorization
	_ = share
	switch urlStatus {
	case UrlFile:
		return conn.serveFile(req, path)
	case UrlDir:
		return conn.serveDirListing(req, path)
	case UrlNotFound:
		return conn.createErrorResponse(req, false, http.StatusNotFound, path)
	case UrlForbidden:
		return conn.createErrorResponse(req, false, http.StatusForbidden, path)
	case UrlIncorrect:
		return conn.createErrorResponse(req, false, http.StatusBadRequest,
			"invalid url '"+req.Url+"'")
	default:
		panic(fmt.Sprintf("unsupported UrlStatus: %d", int32(urlStatus)))
	}
}

func (conn *TcpConnection) serveDirListing(req *http.Request, dirPath string) *http.Response {
	htmlBytes, err := html.GenDirListingHtml(&conn.log, dirPath,
		req.Url, conn.srv.Conf.HtmlTitle)
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

	// time is received with second precision
	// if not truncate, mtime will always be > ifModifiedSince
	mtime := finf.ModTime().UTC().Truncate(time.Second)

	ifModifiedSince, errTime := time.Parse(http.HeaderTimeFormat, req.Headers["if-modified-since"])
	notModified := errTime == nil && !mtime.After(ifModifiedSince)
	if notModified {
		return conn.http.CreateResponse(http.StatusNotModified, req, false)
	}

	res := conn.http.CreateResponse(http.StatusOK, req, false)
	fileName := path.Base(filePath)

	if req.Method != "HEAD" {
		// file will be closed by SendResponse()
		fileOwnershipTransferred = true
		res.WithBodyReader(f, size, http.GetMimeType(fileName))
	}
	// Tell browser to not use its own caching, remember last-modified
	// On next request browser will send if-modified-since and let server check if file changed.
	res.Headers["cache-control"] = "no-cache"
	res.Headers["last-modified"] = mtime.Format(http.HeaderTimeFormat)
	res.Headers["content-disposition"] = http.ContentDisposition("inline", fileName)
	//TODO: accept "range" header and return StatusPartialContent
	// res.Headers["accept-ranges"] = "bytes"

	return res
}
