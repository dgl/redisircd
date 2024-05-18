package http

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var redisHost string

func Start(redis string) {
	redisHost = redis

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/publish/", publishHandler)
}

func Handle(line string, reader io.Reader, writer io.Writer) error {
	req, err := http.ReadRequest(
		bufio.NewReader(io.MultiReader(bytes.NewReader([]byte(line)), reader)))
	if err != nil {
		return err
	}

	http.DefaultServeMux.ServeHTTP(NewClosingResponseWriter(writer), req)
	// Closes connection.
	return errors.New("Handled as HTTP")
}

// ClosingResponseWriter only handles one request, as we don't implement
// keepalive.
type ClosingResponseWriter struct {
	statusCode int
	header     http.Header
	writer     io.Writer
}

func NewClosingResponseWriter(w io.Writer) *ClosingResponseWriter {
	return &ClosingResponseWriter{
		writer: w,
		header: http.Header{
			"Connection": []string{"close"},
		},
	}
}

func (w *ClosingResponseWriter) Header() http.Header {
	return w.header
}

func (w *ClosingResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.writer.Write(b)
}

func (w *ClosingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.writer.Write([]byte(fmt.Sprintf("HTTP/1.0 %d Alright\r\n", statusCode)))
	w.header.Write(w.writer)
	w.writer.Write([]byte("\r\n"))
}
