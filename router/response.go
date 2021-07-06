package router

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/ninlil/butler/log"
)

type ctFormat int

const (
	ctfJSON ctFormat = iota
	ctfXML
)

var ctfText = []string{"json", "xml"}

// ErrUnmarshal is an error when parsing the request-body according to the "Content-Type" header
type ErrUnmarshal ctFormat

func (ctf ErrUnmarshal) Error() string {
	return fmt.Sprintf("unable to parse %s", ctfText[int(ctf)])
}

func (ctf ctFormat) Unmarshal(buf []byte, dest interface{}) error {
	switch ctf {
	case ctfJSON:
		return json.Unmarshal(buf, dest)
	case ctfXML:
		return xml.Unmarshal(buf, dest)
	}
	return nil
}

func getContentTypeFormat(format string) (ctf ctFormat, indent int) {
	ctf = ctfJSON
	indent = 0

	if format != "" {
		media, params, err := mime.ParseMediaType(format)
		if err != nil {
			log.Warn().Msgf("Accept/Content-type - error: %v", err)
		}
		switch media {
		case "application/json":
			ctf = ctfJSON
		case "application/xml":
			ctf = ctfXML
		}
		n, _ := strconv.ParseInt(params["indent"], 0, 0)
		indent = int(n)
	}
	if indent > 10 {
		indent = 10
	}
	return
}

func (rt *Route) writeResponse(w http.ResponseWriter, r *http.Request, status int, data interface{}) {

	ctf, indent := getContentTypeFormat(r.Header.Get("Accept"))

	var ct string
	var buf []byte
	var err error
	switch ctf {
	case ctfJSON:
		ct = ctJSON
		if indent > 0 {
			buf, err = json.MarshalIndent(data, "", strings.Repeat(" ", int(indent)))
		} else {
			buf, err = json.Marshal(data)
		}
	case ctfXML:
		ct = ctXML
		if indent > 0 {
			buf, err = xml.MarshalIndent(data, "", strings.Repeat(" ", int(indent)))
		} else {
			buf, err = xml.Marshal(data)
		}
	}

	if err != nil {
		rt.writeError(err, w, r, http.StatusInternalServerError)
		return
	}

	if indent > 0 {
		ct += fmt.Sprintf("; indent=%d", indent)
	}

	w.Header().Set("Content-Type", ct)
	size := len(buf)
	if size > 0 {
		w.Header().Set("Content-Length", fmt.Sprint(size))
	}

	if status == 0 {
		if size == 0 {
			status = http.StatusNoContent
		} else {
			status = http.StatusOK
		}
	}

	w.WriteHeader(status)
	_, _ = w.Write(buf)
}
