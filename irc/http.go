package irc

import (
	"errors"

	"gopkg.in/sorcix/irc.v2"

	"github.com/dgl/redisircd/http"
)

func (c *Client) maybeHTTP(m *irc.Message) error {
	// Well, this is easier than using cmux? Maybe.

	if len(m.Params) > 0 {
		trail := m.Params[len(m.Params)-1]
		if len(trail) > 7 && trail[:7] == "HTTP/1." {
			// It is HTTP/1.x.
			return http.Handle(c.LastLine(), c.Conn.Reader, c.Conn.Writer)
		}
	}
	return errors.New("Malformed HTTP")
}
