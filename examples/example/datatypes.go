package main

import "time"

type dtArgs struct {
	Int     int           `json:"int" from:"query"`
	Int8    int8          `json:"int8" from:"query"`
	Int16   int16         `json:"int16" from:"query"`
	Int32   int32         `json:"int32" from:"query"`
	Int46   int64         `json:"int64" from:"query"`
	Text    string        `json:"string" from:"query"`
	Float32 float32       `json:"float32" from:"query"`
	Float64 float64       `json:"float64" from:"query"`
	Bool    bool          `json:"bool" from:"query"`
	Time    time.Time     `json:"time" from:"query"`
	Dur     time.Duration `json:"dur" from:"query"`
	Bytes   []byte        `json:"bytes" from:"query"`
}

type dtReturn struct {
	*dtArgs
	BytesAsText string   `json:"bytes_as_text"`
	DurAsText   string   `json:"dur_as_text"`
	XMLName     struct{} `json:"-" xml:"DatatypeArgs"`
}

func types(args *dtArgs) *dtReturn {
	return &dtReturn{
		dtArgs:      args,
		BytesAsText: string(args.Bytes),
		DurAsText:   args.Dur.String(),
	}
}
