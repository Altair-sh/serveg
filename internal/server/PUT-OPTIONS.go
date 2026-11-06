package server

import (
	"errors"
	"fmt"
	"io"
	"os"
	"serveg/internal/http"
	"strconv"
)

func DirOrFileExists(path string) (exists bool, fi os.FileInfo, err error) {
	fi, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	} else if err == nil {
		exists = true
	}
	return
}

var ErrIsUploadingAlready = http.NewHttpError(http.StatusConflict,
	"somebody is already uploading such file")
var ErrFileIsDir = http.NewHttpError(http.StatusConflict,
	"there is a directory with such name already")

// PUT: accept request body and write it to filePath+".uploading"
// OPTIONS: check if file can be uploaded and return without reading body
// returns StatusCreated if upload is a new file, and StatusOK if it is replacing existing file
func (conn *TcpConnection) PUT_OPTIONS(req *http.Request) *http.Response {
	reqHasBody := req.Method == "PUT"
	path, share, urlStatus := conn.srv.Conf.Share.MapUrl(req.Url)
	// handle authorization
	if share != nil {
		if share.WritePassword == "" {
			return conn.createErrorResponse(req, reqHasBody, http.StatusForbidden, "share is read-only")
		}
		access := checkSharePasswords(share, req)
		if access < AccessReadWrite {
			return conn.createErrorResponse(req, reqHasBody,
				http.StatusUnauthorized, "writer password required").
				WithHeader("www-authenticate",
					`Basic realm="`+share.Url+` (write)", charset="UTF-8"`)
		}
	}

	switch urlStatus {
	case UrlFile:
		return conn.handleUpload(req, path, true)
	case UrlDir:
		return conn.http.CreateErrorResponse(req, reqHasBody, ErrFileIsDir)
	case UrlDoesntExistInShare:
		// TODO: allow users to create directories
		return conn.handleUpload(req, path, false)
	case UrlShareNotFound:
		return conn.createErrorResponse(req, reqHasBody, http.StatusNotFound, req.Url)
	case UrlForbidden:
		return conn.createErrorResponse(req, reqHasBody, http.StatusForbidden, req.Url)
	case UrlIncorrect:
		return conn.createErrorResponse(req, reqHasBody, http.StatusBadRequest,
			"invalid url '"+req.Url+"'")
	default:
		panic(fmt.Sprintf("unsupported UrlStatus: %d", int32(urlStatus)))
	}
}

func (conn *TcpConnection) handleUpload(req *http.Request, filePath string, fileExists bool,
) *http.Response {
	reqHasBody := req.Method == "PUT"
	maxUpload := conn.srv.Conf.MaxUpload
	var resultStatus http.StatusCode
	if fileExists {
		resultStatus = http.StatusOK
	} else {
		resultStatus = http.StatusCreated
	}

	errToResponse := func(err error) *http.Response {
		httpErr, ok := errors.AsType[*http.HttpError](err)
		if !ok {
			httpErr = http.NewHttpError(http.StatusInternalServerError, err.Error())
		}
		return conn.http.CreateErrorResponse(req, reqHasBody, httpErr)
	}
	createResponseMsg := func(status http.StatusCode, msg string) *http.Response {
		return conn.http.CreateResponse(status, req, false).
			WithBodyStr(msg, http.TextPlain)
	}

	tmpPath := filePath + ".uploading"
	tmpFileExists, fi, err := DirOrFileExists(tmpPath)
	if err != nil {
		return errToResponse(err)
	}
	if tmpFileExists {
		var httpErr *http.HttpError
		if fi.IsDir() {
			httpErr = ErrFileIsDir
		} else {
			httpErr = ErrIsUploadingAlready
		}
		return conn.http.CreateErrorResponse(req, reqHasBody, httpErr)
	}

	// OPTIONS
	if !reqHasBody {
		uploadSize, err := strconv.ParseUint(req.Headers["x-upload-size"], 10, 64)
		if err != nil {
			return createResponseMsg(http.StatusBadRequest,
				"OPTIONS request must have valid header x-upload-size")
		}
		if uploadSize > maxUpload {
			return createResponseMsg(http.StatusRequestEntityTooLarge,
				fmt.Sprintf("x-upload-size (%v) > maxUpload (%v)",
					uploadSize, maxUpload))
		}
		return conn.http.CreateResponse(resultStatus, req, false)
	}

	// PUT
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		return errToResponse(ErrIsUploadingAlready)
	}
	if err != nil {
		return errToResponse(err)
	}
	renamed := false
	defer func() {
		_ = f.Close()
		// remove temp file if upload failed
		if !renamed {
			// TODO: log this error
			_ = os.Remove(tmpPath)
		}
	}()

	// read request body
	reqBody, err := req.GetBodyReader(int64(maxUpload))
	if err != nil {
		return errToResponse(err)
	}
	n, err := io.Copy(f, reqBody)
	if err != nil {
		return errToResponse(err)
	}
	if n != req.ContentLength {
		return conn.createErrorResponse(req, true, http.StatusBadRequest,
			"uploaded file size != content-length")
	}

	// move tmpPath to filePath
	if err = f.Sync(); err != nil {
		return errToResponse(err)
	}
	if err = f.Close(); err != nil {
		return errToResponse(err)
	}
	if err = os.Rename(tmpPath, filePath); err != nil {
		return errToResponse(err)
	}
	renamed = true

	// TODO: is it right status?
	return conn.http.CreateResponse(resultStatus, req, false)
}
