package irc

import (
	"log"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
	"strings"

	"gopkg.in/sorcix/irc.v2"
)

type CommandFn func(*Client, *irc.Message) error

type commandMap map[string]CommandFn

var commands = commandMap{
	"PING": (*Client).ping,
	"PONG": (*Client).pong,
	"JOIN": (*Client).join,
	"QUIT": (*Client).userQuit,
	"PART": (*Client).part,
	"PRIVMSG": (*Client).msg,
	"NOTICE": (*Client).msg,
}

// commands receives inbound commands from the client
func (c *Client) commands() error {
	for {
		c.tcpConn.SetReadDeadline(time.Now().Add(timeoutDuration))
		message, err := c.Decode()
		if err != nil {
			if oerr, ok := err.(*net.OpError); ok {
				if oerr.Timeout() {
					// Timeout, maybe send a ping?
					now := time.Now()
					if c.last.Add(3*timeoutDuration).After(now) {
						c.Encode(&irc.Message{Command: "PING", Params: []string{c.Server.Name}})
					} else {
						return c.quit(fmt.Sprintf("Ping timeout (%d seconds)", int(now.Sub(c.last)/time.Second)))
					}
					continue
				} else {
					return c.quit(oerr.Err.Error())
				}
			}
			if err == io.EOF {
				return c.quit(err.Error())
			}
			log.Printf("Closing connection (%q): %v", c.User.Prefix, err)
			// Some kind of unexpected error, we might leak unexpected info in the
			// quit message if we use it.
			return c.quit("ERROR")
		}
		if message == nil {
			continue
		}
		c.last = time.Now()
		log.Print(message)

		if cmd, ok := commands[message.Command]; ok {
			err := cmd(c, message)
			if err != nil {
				return err
			}
		} else {
			c.reply(irc.ERR_UNKNOWNCOMMAND, "Unknown command")
		}
	}
}

func (c *Client) ping(m *irc.Message) error {
	if len(m.Params) < 1 {
		c.reply(irc.ERR_NEEDMOREPARAMS, "PING", "Not enough parameters")
		return nil
	}

	c.Encode(&irc.Message{
		Prefix: &irc.Prefix{Name: c.Server.Name},
		Command: "PONG",
		Params: []string{c.Server.Name, m.Params[0]}})

	return nil
}

func (c *Client) pong(m *irc.Message) error {
	return nil
}

func (c *Client) join(m *irc.Message) error {
	if len(m.Params) < 1 {
		c.reply(irc.ERR_NEEDMOREPARAMS, "JOIN", "Not enough parameters")
		return nil
	}

	for _, ch := range strings.Split(m.Params[0], ",") {
		if !validChan(ch) {
			c.reply("479" /* Not in RFC2812, but used by various IRCd */, ch, "Illegal channel name")
			return nil
		}
		c.Server.cs.send(chanRequest{Name: ch, Type: CR_JOIN, User: c.User})
	}
	return nil
}

func (c *Client) msg(m *irc.Message) error {
	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		c.reply(irc.ERR_NORECIPIENT, "No recipient given")
		return nil
	}
	if len(m.Params) < 2 || len(m.Params[1]) < 1 {
		c.reply(irc.ERR_NOTEXTTOSEND, "No text to send")
		return nil
	}

	t := chanReqType(CR_PRIVMSG)
	if m.Command == "NOTICE" {
		t = CR_NOTICE
	}

	target := m.Params[0]
	text := m.Params[1]
	if target[0] == '#' {
		c.Server.cs.send(chanRequest{Name: target, Type: t, User: c.User, Params: []string{text}})
	} else {
		// nick...
	}

	return nil
}

func (c *Client) part(m *irc.Message) error {
	if len(m.Params) < 1 {
		c.reply(irc.ERR_NEEDMOREPARAMS, "PART", "Not enough parameters")
		return nil
	}

	target := m.Params[0]
	reason := ""
	if len(m.Params) >= 2 {
		reason = m.Params[1]
	}

	if !validChan(target) {
		c.reply("479" /* Not in RFC2812, but used by various IRCd */, target, "Illegal channel name")
		return nil
	}

	c.Server.cs.send(chanRequest{Name: target, Type: CR_LEAVE, User: c.User, Params: []string{reason}})
	return nil
}

func (c *Client) userQuit(m *irc.Message) error {
	reason := ""
	if len(m.Params) >= 1 {
		reason = m.Params[0]
	}

	if len(reason) > 0 {
		return c.quit("\"" + reason + "\"")
	} else {
		return c.quit("")
	}
}

func (c *Client) quit(reason string) error {
	c.Server.cs.send(chanRequest{Type: CR_QUIT, User: c.User, Params: []string{reason}})
	c.Server.ns.send(nickRequest{Type: NR_QUIT, Name: c.User.Nick})
	c.Encode(&irc.Message{
		Prefix: c.User.Prefix,
		Command: "QUIT",
		Params: []string{reason}})
	return errors.New(reason)
}
