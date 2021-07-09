package router

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type fromSource int

const (
	fromPath fromSource = iota + 1
	fromQuery
	fromHeader
	fromBody
)

type tagInfo struct {
	Name       string
	From       fromSource
	Required   bool
	HasMin     bool
	HasMax     bool
	HasDefault bool
	hasRegex   bool
	Min        string
	Max        string
	Default    string
	Regex      string
}

func parseTag(st reflect.StructTag) *tagInfo {
	var tags tagInfo

	tags.Name = st.Get("json")

	from := st.Get("from")
	switch true {
	case from == "header":
		tags.From = fromHeader
	case from == "query":
		tags.From = fromQuery
	case from == "body":
		tags.From = fromBody
	default:
		tags.From = fromPath
	}

	var txt string
	txt, tags.Required = st.Lookup("required")
	if flag, err := strconv.ParseBool(txt); err == nil && txt != "" {
		tags.Required = flag
	}

	tags.Min, tags.HasMin = st.Lookup("min")
	tags.Max, tags.HasMax = st.Lookup("max")
	tags.Default, tags.HasDefault = st.Lookup("default")
	tags.Regex, tags.hasRegex = st.Lookup("regex")

	return &tags
}

func (tag *tagInfo) int(f reflect.Value, txt string, force bool) error {
	v, err := strconv.ParseInt(txt, 0, 0)
	// log.Debug().Msgf("field.int: %s -> %d (%v)", txt, v, err)
	if err != nil {
		return err
	}

	if !force {
		if tag.HasMin {
			min, err := strconv.ParseInt(tag.Min, 0, 0)
			if err != nil {
				return err
			}
			if v < min {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgBelowMin, min))
			}
		}

		if tag.HasMax {
			max, err := strconv.ParseInt(tag.Max, 0, 0)
			if err != nil {
				return err
			}
			if v > max {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgAboveMax, max))
			}
		}
	}

	f.SetInt(v)
	return nil
}

func (tag *tagInfo) float(f reflect.Value, txt string, force bool) error {
	v, err := strconv.ParseFloat(txt, 0)
	// log.Debug().Msgf("field.float: %s -> %g (%v)", txt, v, err)
	if err != nil {
		return err
	}

	if !force {
		if tag.HasMin {
			min, err := strconv.ParseFloat(tag.Min, 0)
			if err != nil {
				return err
			}
			if v < min {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgBelowMin, min))
			}
		}

		if tag.HasMax {
			max, err := strconv.ParseFloat(tag.Max, 0)
			if err != nil {
				return err
			}
			if v > max {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgAboveMax, max))
			}
		}
	}

	f.SetFloat(v)
	return nil
}
func (tag *tagInfo) duration(f reflect.Value, txt string, force bool) error {
	v, err := time.ParseDuration(txt)
	// log.Debug().Msgf("field.duration: %s -> %v (%v)", txt, v, err)
	if err != nil {
		return err
	}

	if !force {
		if tag.HasMin {
			min, err := time.ParseDuration(tag.Min)
			if err != nil {
				return err
			}
			if v < min {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgBelowMin, min))
			}
		}

		if tag.HasMax {
			max, err := time.ParseDuration(tag.Max)
			if err != nil {
				return err
			}
			if v > max {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgAboveMax, max))
			}
		}
	}

	f.SetInt(int64(v))
	return nil
}

func (tag *tagInfo) bool(f reflect.Value, txt string, _ bool) error {
	v, err := strconv.ParseBool(txt)
	// log.Debug().Msgf("field.bool: %s -> %t (%v)", txt, v, err)
	if err != nil {
		return err
	}

	f.SetBool(v)
	return nil
}

func (tag *tagInfo) string(f reflect.Value, txt string, force bool) error {
	// log.Debug().Msgf("field.string: %s", txt)

	if !force {
		if tag.HasMin {
			min, err := strconv.ParseInt(tag.Min, 0, 0)
			if err != nil {
				return err
			}
			if int64(len(txt)) < min {
				return newFieldError(nil, tag.Name, txt, fmt.Sprintf(errMsgBelowMin, min))
			}
		}

		if tag.HasMax {
			max, err := strconv.ParseInt(tag.Max, 0, 0)
			if err != nil {
				return err
			}
			if int64(len(txt)) < max {
				return newFieldError(nil, tag.Name, txt, fmt.Sprintf(errMsgAboveMax, max))
			}
		}
	}

	f.SetString(txt)
	return nil
}

func (tag *tagInfo) bytes(f reflect.Value, txt string, force bool) error {
	// log.Debug().Msgf("field.bytes: %s", txt)
	buf, err := base64.StdEncoding.DecodeString(txt)
	if err != nil {
		return err
	}

	if !force {
		if tag.HasMin {
			min, err := strconv.ParseInt(tag.Min, 0, 0)
			if err != nil {
				return err
			}
			if int64(len(buf)) < min {
				return newFieldError(nil, tag.Name, txt, fmt.Sprintf(errMsgBelowMin, min))
			}
		}

		if tag.HasMax {
			max, err := strconv.ParseInt(tag.Max, 0, 0)
			if err != nil {
				return err
			}
			if int64(len(buf)) < max {
				return newFieldError(nil, tag.Name, txt, fmt.Sprintf(errMsgAboveMax, max))
			}
		}
	}

	f.Set(reflect.ValueOf(buf))
	return nil
}

func (tag *tagInfo) time(f reflect.Value, txt string, force bool) error {
	v, err := parseTime(txt)
	// log.Debug().Msgf("field.float: %s -> %v (%v)", txt, v, err)
	if err != nil {
		return err
	}

	if !force {
		if tag.HasMin {
			min, err := parseTime(tag.Min)
			if err != nil {
				return err
			}
			if v.Before(min) {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgBelowMin, min))
			}
		}

		if tag.HasMax {
			max, err := parseTime(tag.Max)
			if err != nil {
				return err
			}
			if v.After(max) {
				return newFieldError(nil, tag.Name, v, fmt.Sprintf(errMsgAboveMax, max))
			}
		}
	}

	f.Set(reflect.ValueOf(v))
	return nil
}

func parseTime(txt string) (time.Time, error) {
	if dt, err := time.Parse(time.RFC3339Nano, txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse(time.RFC3339, txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse(time.RFC822Z, txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse(time.RFC822, txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse(time.RFC1123Z, txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse(time.RFC1123, txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse("2006-01-02", txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse("2006-01-02 15:04:05.99999", txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse("2006-01-02 15:04:05", txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse("15:04:05.99999", txt); err == nil {
		return dt, nil
	}
	if dt, err := time.Parse("15:04:05", txt); err == nil {
		return dt, nil
	}
	return time.Time{}, fmt.Errorf("unable to parse date/time")
}
