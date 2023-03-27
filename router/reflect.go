package router

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/ninlil/butler/log"
)

type paramData struct {
	ptr   reflect.Value
	data  reflect.Value
	dt    reflect.Type
	vars  map[string]string
	query url.Values
}

func (rt *Route) createArgs(w http.ResponseWriter, r *http.Request) ([]reflect.Value, error) {
	nArgs := rt.fnType.NumIn()
	args := make([]reflect.Value, nArgs)

	for i := 0; i < nArgs; i++ {
		arg := rt.fnType.In(i)

		// log.Debug().Msgf("arg #%d is a %s", i, arg.Kind())

		switch true {
		case arg == tContext:
			args[i] = reflect.ValueOf(r.Context())

		case arg == tResponseWriter:
			args[i] = reflect.ValueOf(w)

		case arg == tRequest:
			args[i] = reflect.ValueOf(r)

		case arg.Kind() == reflect.Struct:
			ptr, err := rt.createStruct(arg, r)
			if err != nil {
				return nil, err
			}
			args[i] = ptr.Elem()

		case arg.Kind() == reflect.Ptr && arg.Elem().Kind() == reflect.Struct:
			ptr, err := rt.createStruct(arg.Elem(), r)
			if err != nil {
				return nil, err
			}
			args[i] = ptr

		default:
			log.Warn().Msgf("route %s: arg %d is of unknown type: %v", rt.Name, i, arg.String())
		}
	}

	return args, nil
}

func (param *paramData) getValue(f reflect.Value, tags *tagInfo, r *http.Request) (value string, found bool, handled bool, err error) {
	switch tags.From {
	case fromPath:
		if param.vars == nil {
			params := chi.RouteContext(r.Context()).URLParams
			param.vars = make(map[string]string, len(params.Keys))
			for i, key := range params.Keys {
				param.vars[key] = params.Values[i]
			}
		}
		value, found = param.vars[tags.Name]

	case fromHeader:
		value = r.Header.Get(tags.Name)
		found = value != ""

	case fromQuery:
		if param.query == nil {
			param.query = r.URL.Query()
		}
		var values []string
		values, found = param.query[tags.Name]
		if found && len(values) > 0 {
			value = values[0]
		}

	case fromBody:
		return getBodyValue(f, r)

	default:
		panic("illegal 'from'")
	}

	if tags.hasRegex && found {
		var re *regexp.Regexp
		re, err = getRegexp(tags.Regex)
		if err != nil {
			log := log.FromCtx(r.Context())
			log.Warn().Msgf("field '%s' has invalid regex '%s': %v", tags.Name, tags.Regex, err)
			return
		}
		log.Debug().Msgf("? regexp-match '%s' with '%s' == %t", value, tags.Regex, re.MatchString(value))
		if !re.MatchString(value) {
			err = ErrInvalidMatch
			return
		}
	}
	return
}

func getBodyValue(f reflect.Value, r *http.Request) (value string, found bool, handled bool, err error) {
	var raw []byte
	raw, err = io.ReadAll(r.Body)
	size := len(raw)
	found = err == nil && size > 0

	if found {
		ft := f.Type()

		if ft.Kind() == reflect.String {
			f.Set(reflect.ValueOf(string(raw)))
			return "", true, true, nil
		}

		if f.Kind() == reflect.Slice {
			e := ft.Elem()

			switch e.Kind() {
			case reflect.Uint8:
				f.Set(reflect.ValueOf(raw))

			case reflect.String:
				buf := bytes.NewBuffer(raw)
				scanner := bufio.NewScanner(buf)
				var lines []string
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				f.Set(reflect.ValueOf(lines))

			default:
				log.Warn().Msgf("Body is an array of unknown type: %s", e.String())
				return string(value), false, false, nil
			}
			return "", true, true, nil
		}

		var body interface{}

		if f.Kind() == reflect.Struct {
			body = f.Addr().Interface()
		} else {
			body = reflect.New(f.Type().Elem()).Interface()
		}

		ctf, _, _ := getContentTypeFormat(r.Header.Get("Content-Type"))
		err = ctf.Unmarshal(raw, body)
		if err != nil {
			return
		}
		if f.Kind() == reflect.Ptr {
			f.Set(reflect.ValueOf(body))
		}
	}
	handled = true
	return
}

func (param *paramData) fillField(i int, r *http.Request) error {
	var value string
	// var raw []byte
	var found, isDefault bool
	var err error
	// var query url.Values

	f := param.data.Field(i)
	tags := parseTag(param.dt.Field(i).Tag)

	value, found, handled, err := param.getValue(f, tags, r)
	if handled || err != nil {
		if err != nil {
			err = newFieldError(err, tags.Name, value, err.Error())
		}
		return err
	}

	if !found && tags.HasDefault {
		value = tags.Default
		found = true
		isDefault = true
	}

	// log.Debug().Msgf("field %d is a %s - v:%s found:%t", i, f.Kind(), value, found)

	if tags.Required && !found {
		return newFieldError(nil, tags.Name, nil, errMsgRequired)
	}

	if found {
		err = param.assignField(f, value, isDefault, tags)
		if err != nil {
			return newFieldError(err, tags.Name, value, err.Error())
		}
	}

	return nil
}

func (param *paramData) assignField(f reflect.Value, value string, isDefault bool, tags *tagInfo) error {
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if f.Type() == tDur {
			return tags.duration(f, value, isDefault)
		}
		return tags.int(f, value, isDefault)

	case reflect.Float32, reflect.Float64:
		return tags.float(f, value, isDefault)

	case reflect.String:
		return tags.string(f, value, isDefault)

	case reflect.Bool:
		return tags.bool(f, value, isDefault)

	case reflect.Struct:
		if f.Type() == tTime {
			return tags.time(f, value, isDefault)
		}
		return fmt.Errorf("struct not recognized")

	case reflect.Slice:
		switch f.Type().Elem().Kind() {
		case reflect.Uint8: // bytes
			return tags.bytes(f, value, isDefault)
		default:
			return fmt.Errorf("slice of %s not supported", f.Type().Elem().Kind())
		}

	default:
		return newFieldError(nil, tags.Name, value, fmt.Sprintf(errMsgUnknownType, f.Kind()))
	}
}

func (rt *Route) createStruct(arg reflect.Type, r *http.Request) (reflect.Value, error) {

	param := &paramData{
		ptr: reflect.New(arg),
	}
	param.data = param.ptr.Elem()
	param.dt = param.data.Type()

	n := param.data.NumField()
	for i := 0; i < n; i++ {
		err := param.fillField(i, r)
		if err != nil {
			return param.ptr, err
		}
	}

	// log.Debug().Msgf("createStruct: %+v", param.ptr)

	return param.ptr, nil
}
