package bufferedresponse

import (
	"bytes"
	"net/http"
	"strconv"
)

// ResponseWriter acts as a buffered http.ResponseWriter
type ResponseWriter struct {
	rw     http.ResponseWriter
	buffer bytes.Buffer
	status int
	sent   bool
}

// Wrap a regular http.ResponseWriter in a buffered version
func Wrap(rw http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		rw:     rw,
		buffer: bytes.Buffer{},
		status: http.StatusOK,
	}
}

// Get the buffered version from the normal type (if possible)
func Get(rw http.ResponseWriter) (*ResponseWriter, bool) {
	rw2, ok := rw.(*ResponseWriter)
	return rw2, ok
}

// Header allows for editing the http-headers sent with the response
func (rw *ResponseWriter) Header() http.Header {
	return rw.rw.Header()
}

// Write adds content (body) to the the response, appends to already written data
func (rw *ResponseWriter) Write(buf []byte) (int, error) {
	if rw.sent {
		return rw.rw.Write(buf)
	}
	return rw.buffer.Write(buf)
}

// WriteHeader sets the status-code of the return-value (default = http.StatusOK)
func (rw *ResponseWriter) WriteHeader(status int) {
	if rw.sent {
		panic("response already sent, unable to change status")
	}
	rw.status = status
}

// Flush sends all headers, status and body
func (rw *ResponseWriter) Flush() {
	if rw.sent {
		return
	}
	rw.rw.WriteHeader(rw.status)
	_, _ = rw.rw.Write(rw.buffer.Bytes())
	rw.sent = true
}

// Reset the content
func (rw *ResponseWriter) Reset() {
	rw.buffer.Reset()
}

// Size returns the content size in bytes
func (rw *ResponseWriter) Size() int {
	return rw.buffer.Len()
}

// Status returns the current result-status
func (rw *ResponseWriter) Status() int {
	return rw.status
}

// SetContentLength sets the 'Content-Length' header to the current size
func (rw *ResponseWriter) SetContentLength() {
	rw.Header().Set("Content-Length", strconv.Itoa(rw.Size()))
}
