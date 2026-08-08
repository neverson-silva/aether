package core

import (
	"bytes"
	"compress/gzip"
	"io"
)

func gzipWriter(w io.Writer) *gzip.Writer {
	return gzip.NewWriter(w)
}

func gunzipBytes(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
}

func ioCopy(w io.Writer, r io.Reader) (int64, error) {
	return io.Copy(w, r)
}
