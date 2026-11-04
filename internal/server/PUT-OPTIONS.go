package server

import "serveg/internal/http"

// TODO implement file uploading
func (conn *TcpConnection) handleUpload(req *http.Request) *http.Response {
	// TODO: use os.OpenFile(filePath + ".upload", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	// if such file exists, send response like
	// "somebody is uploading such file right now"
	// TODO: defer { if !renamed { os.Remove(tmp) } }
	panic("unimplemented")
}
