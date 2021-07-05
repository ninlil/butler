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

func (rt *Route) writeResponse(w http.ResponseWriter, r *http.Request, status int, data interface{}) {

	var indent int64
	ctf := ctfJSON
	format := r.Header.Get("Accept")
	if format != "" {
		media, params, err := mime.ParseMediaType(format)
		if err == nil {
			log.Warn().Msgf("Accept-header-error: %v", err)
		}
		switch media {
		case "application/json":
			ctf = ctfJSON
		case "application/xml":
			ctf = ctfXML
		}
		indent, _ = strconv.ParseInt(params["indent"], 0, 0)
	}
	if indent > 10 {
		indent = 10
	}

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
