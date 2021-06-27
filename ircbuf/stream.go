// Copyright 2014 Vic Demuzere
//
// Use of this source code is governed by the MIT license.

// Modified from irc.v2 to make the buffer public and remove locking (we always
// use from the same goroutine), also add writev().

package ircbuf

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"

	"gopkg.in/sorcix/irc.v2"
)

// Messages are delimited with CR and LF line endings,
// we're using the last one to split the stream. Both are removed
// during message parsing.
const delim byte = '\n'

var endline = []byte("\r\n")

// A Conn represents an IRC network protocol connection.
// It consists of an Encoder and Decoder to manage I/O.
type Conn struct {
	Encoder
	Decoder

	conn io.ReadWriteCloser
}

// NewConn returns a new Conn using rwc for I/O.
func NewConn(rwc io.ReadWriteCloser) *Conn {
	return &Conn{
		Encoder: Encoder{
			Writer: rwc,
		},
		Decoder: Decoder{
			Reader: bufio.NewReader(rwc),
		},
		conn: rwc,
	}
}

// Dial connects to the given address using net.Dial and
// then returns a new Conn for the connection.
func Dial(addr string) (*Conn, error) {
	c, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return NewConn(c), nil
}

// DialTLS connects to the given address using tls.Dial and
// then returns a new Conn for the connection.
func DialTLS(addr string, config *tls.Config) (*Conn, error) {
	c, err := tls.Dial("tcp", addr, config)

	if err != nil {
		return nil, err
	}

	return NewConn(c), nil
}

// Close closes the underlying ReadWriteCloser.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// A Decoder reads Message objects from an input stream.
type Decoder struct {
	Reader *bufio.Reader
	line   string
}

// NewDecoder returns a new Decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		Reader: bufio.NewReader(r),
	}
}

// Decode attempts to read a single Message from the stream.
//
// Returns a non-nil error if the read failed.
func (dec *Decoder) Decode() (m *irc.Message, err error) {

	dec.line, err = dec.Reader.ReadString(delim)

	if err != nil {
		return nil, err
	}

	return irc.ParseMessage(dec.line), nil
}

// LastLine returns the last line read by Decoder, in raw form.
func (dec *Decoder) LastLine() string {
	return dec.line
}

// An Encoder writes Message objects to an output stream.
type Encoder struct {
	Writer io.Writer
}

// NewEncoder returns a new Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		Writer: w,
	}
}

// Encode writes the IRC encoding of m to the stream.
//
// Returns an non-nil error if the write to the underlying stream stopped early.
func (enc *Encoder) Encode(m *irc.Message) (err error) {

	_, err = enc.Write(m.Bytes())

	return
}

// Write writes len(p) bytes from p followed by CR+LF.
func (enc *Encoder) Write(p []byte) (n int, err error) {

	if tcpconn, ok := enc.Writer.(*net.TCPConn); ok {
		buffers := net.Buffers{p, endline}

		var nv int64
		nv, err = buffers.WriteTo(tcpconn)
		if err != nil {
			// Truncation ok; limited by IRC line length anyway.
			n = int(nv)
		}
	} else {
		n, err = enc.Writer.Write(p)
		if err != nil {
			return
		}

		_, err = enc.Writer.Write(endline)
	}
	return
}
