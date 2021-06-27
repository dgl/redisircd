package irc

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gopkg.in/sorcix/irc.v2"
)

const (
	timeoutDuration = 30 * time.Second
)

var preCommands = commandMap{
	"QUIT": (*Client).preQuit,
	"USER": (*Client).preUser,
	"NICK": (*Client).preNick,
	"PING": (*Client).ping,
}

func (c *Client) pre() error {
	for {
		c.tcpConn.SetReadDeadline(time.Now().Add(timeoutDuration))
		message, err := c.Decode()
		if err != nil || message == nil {
			log.Printf("Decode: %v", err)
			return err
		}
		c.last = time.Now()
		log.Print(message)

		if cmd, ok := preCommands[message.Command]; ok {
			err := cmd(c, message)
			if err != nil {
				return err
			}
		} else if _, ok := commands[message.Command]; ok {
			c.reply(irc.ERR_NOTREGISTERED, "Command not yet available")
		} else {
			c.reply(irc.ERR_UNKNOWNCOMMAND, "Unknown command")
		}

		if c.User != nil {
			return nil
		}
	}
}

func (c *Client) reply(numeric string, params ...string) {
	nick := "*"
	if len(c.nick) > 0 {
		nick = c.nick
	}
	c.Encode(&irc.Message{
		Prefix:  &irc.Prefix{Name: c.Server.Name},
		Command: numeric,
		Params:  append([]string{nick}, params...)})
}

func (c *Client) preQuit(m *irc.Message) error {
	message := ""
	if len(m.Params) > 0 {
		message = m.Params[0]
	}
	c.Encode(&irc.Message{
		Prefix:  &irc.Prefix{Name: c.nick, User: c.username, Host: "0.0.0.0"},
		Command: "QUIT",
		Params:  []string{message}})
	return errors.New("QUIT :" + message)
}

func (c *Client) preUser(m *irc.Message) error {
	if len(m.Params) < 4 {
		c.reply(irc.ERR_NEEDMOREPARAMS, "USER", "Not enough parameters")
		return nil
	}

	// TODO: Validate user
	c.username = "~" + m.Params[0]
	// TODO: Truncate realname?
	c.Realname = m.Params[3]

	if len(c.nick) > 0 {
		c.connect()
	}
	return nil
}

func (c *Client) preNick(m *irc.Message) error {
	if len(m.Params) < 1 {
		c.reply(irc.ERR_NONICKNAMEGIVEN, "No nickname given")
		return nil
	}

	if !validNick(m.Params[0]) {
		c.reply(irc.ERR_ERRONEUSNICKNAME, "Bad nickname")
		return nil
	}

	c.nick = m.Params[0]

	if len(c.Realname) > 0 {
		c.connect()
	}
	return nil
}

func (c *Client) connect() {
	req := nickRequest{
		Type:   NR_NEW,
		Name:   c.nick,
		Client: c,
		Reply:  make(chan *User),
	}
	c.Server.ns.send(req)

	u := <-req.Reply
	if u == nil {
		n := c.nick
		c.nick = "*"
		c.reply(irc.ERR_NICKNAMEINUSE, n, "Nickname already in use")
		return
	}
	c.User = u

	c.reply(irc.RPL_WELCOME, fmt.Sprintf("Welcome to something like IRC, %s", c.nick))
	c.reply(irc.RPL_YOURHOST, fmt.Sprintf("Your host is %s, running version redisircd-0.XXX", c.Server.Name))
	c.reply(irc.RPL_MYINFO, c.Server.Name, "redisircd-0.XXX", "iw", "noR", "oR")
	// TODO: 005 / etc
}
