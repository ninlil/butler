package router

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/ninlil/butler/bufferedresponse"
	"github.com/ninlil/butler/log"
)

type ctFormat int

const (
	ctfJSON ctFormat = iota
	ctfXML
	ctfTEXT
)

var ctfName = []string{"json", "xml", "text"}

// ErrUnmarshal is an error when parsing the request-body according to the "Content-Type" header
type ErrUnmarshal ctFormat

func (ctf ErrUnmarshal) Error() string {
	return fmt.Sprintf("unable to parse %s", ctfName[int(ctf)])
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

func getContentTypeFormat(format string) (ctf ctFormat, indent int, isCustom bool) {
	ctf = ctfJSON
	indent = 0

	if format != "" {
		media, params, err := mime.ParseMediaType(format)
		if err != nil {
			log.Warn().Msgf("Accept/Content-type - error: %v", err)
		}
		switch media {
		case "application/json":
			isCustom = true
			ctf = ctfJSON
		case "application/xml":
			isCustom = true
			ctf = ctfXML
		case "text/plain":
			isCustom = true
			ctf = ctfTEXT
		}
		n, _ := strconv.ParseInt(params["indent"], 0, 0)
		indent = int(n)
	}
	if indent > 10 {
		indent = 10
	}
	return
}

func createResponse(accept string, data interface{}) (buf []byte, ct string, indent int, err error) {
	ctf, indent, isCustom := getContentTypeFormat(accept)

	if tmp, ok := data.([]byte); ok {
		buf = tmp
		return
	}

	if isCustom || data != nil {
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

		case ctfTEXT:
			ct = ctTEXT
			switch o := data.(type) {
			case string:
				buf = []byte(o)
			case []string:
				buf = []byte(strings.Join(o, "\n"))
			}
		}
	}
	return
}

func (rt *Route) writeResponse(w http.ResponseWriter, r *http.Request, status int, data interface{}) {

	var w2 *bufferedresponse.ResponseWriter = nil
	if data == nil {
		w2, _ = bufferedresponse.Get(w)
	}

	buf, ct, indent, err := createResponse(r.Header.Get("Accept"), data)

	if err != nil {
		rt.writeError(err, w, r, http.StatusInternalServerError)
		return
	}

	if indent > 0 {
		ct += fmt.Sprintf("; indent=%d", indent)
	}

	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	size := len(buf)
	if w2 != nil {
		size += w2.Size()
	}

	if size > 0 {
		w.Header().Set("Content-Length", fmt.Sprint(size))
	}

	if status == 0 {
		if (size == 0) && !rt.router.skip204 {
			status = http.StatusNoContent
		} else {
			status = http.StatusOK
		}
	}

	w.WriteHeader(status)
	if len(buf) > 0 {
		_, _ = w.Write(buf)
	}
}
