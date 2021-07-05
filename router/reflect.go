package router

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// var zero = reflect.Zero(reflect.TypeOf(""))

func (rt *Route) createArgs(w http.ResponseWriter, r *http.Request) ([]reflect.Value, error) {
	nArgs := rt.fnType.NumIn()
	args := make([]reflect.Value, nArgs)

	for i := 0; i < nArgs; i++ {
		arg := rt.fnType.In(i)

		log.Debug().Msgf("arg #%d is a %s", i, arg.Kind())

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
		}
	}

	return args, nil
}

func (rt *Route) createStruct(arg reflect.Type, r *http.Request) (reflect.Value, error) {
	ptr := reflect.New(arg)
	data := ptr.Elem()
	dt := data.Type()

	n := data.NumField()
	for i := 0; i < n; i++ {
		f := data.Field(i)
		tags := parseTag(dt.Field(i).Tag)

		var value string
		var raw []byte
		var found, isDefault bool
		var err error
		var vars map[string]string
		var query url.Values

		switch tags.From {
		case fromPath:
			if vars == nil {
				vars = mux.Vars(r)
			}
			value, found = vars[tags.Name]

		case fromHeader:
			value = r.Header.Get(tags.Name)
			found = value != ""

		case fromQuery:
			if query == nil {
				query = r.URL.Query()
			}
			var values []string
			values, found = query[tags.Name]
			if found && len(values) > 0 {
				value = values[0]
			}

		case fromBody:
			raw, err = ioutil.ReadAll(r.Body)
			found = err == nil && len(raw) > 0

		default:
			panic("illegal 'from'")
		}

		if !found && tags.HasDefault {
			value = tags.Default
			found = true
			isDefault = true
		}

		log.Debug().Msgf("field %d is a %s - v:%s found:%t", i, f.Kind(), value, found)

		if tags.Required && !found {
			// TODO
			return ptr, newFieldError(tags.Name, nil, errMsgRequired)
		}

		if found {
			switch f.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if f.Type() == tDur {
					err = tags.duration(f, value, isDefault)
				} else {
					err = tags.int(f, value, isDefault)
				}

			case reflect.Float32, reflect.Float64:
				err = tags.float(f, value, isDefault)

			case reflect.String:
				err = tags.string(f, value, isDefault)

			case reflect.Bool:
				err = tags.bool(f, value, isDefault)

			case reflect.Struct:
				if f.Type() == tTime {
					err = tags.time(f, value, isDefault)
				} else {
					err = fmt.Errorf("struct not recognized")
				}

			case reflect.Slice:
				switch f.Type().Elem().Kind() {
				case reflect.Uint8: // bytes
					err = tags.bytes(f, value, isDefault)
				default:
					err = fmt.Errorf("slice of %s not supported", f.Type().Elem().Kind())
				}

			default:
				err = newFieldError(tags.Name, value, fmt.Sprintf(errMsgUnknownType, f.Kind()))
			}
			if err != nil {
				if _, ok := err.(FieldError); !ok {
					err = newFieldError(tags.Name, value, err.Error())
				}
				return ptr, err
			}
		}
	}

	log.Debug().Msgf("createStruct: %+v", ptr)

	return ptr, nil
}
