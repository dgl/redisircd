package irc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/sorcix/irc.v2"
)

func init() {
	// TODO: Move to proper startup
	http.Handle("/metrics", promhttp.Handler())
}

func (c *Client) maybeHTTP(m *irc.Message) error {
	// Well, this is easier than using cmux? Maybe.

	if len(m.Params) > 0 {
		trail := m.Params[len(m.Params)-1]
		if len(trail) > 7 && trail[:7] == "HTTP/1." {
			// It is HTTP/1.x.
			req, err := http.ReadRequest(
				bufio.NewReader(io.MultiReader(bytes.NewReader([]byte(c.LastLine())), c.Conn.Reader)))
			if err != nil {
				return err
			}

			http.DefaultServeMux.ServeHTTP(NewClosingResponseWriter(c.Conn.Writer), req)
			// Closes connection.
			return errors.New("Handled as HTTP")
		}
	}
	return errors.New("Malformed HTTP")
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
