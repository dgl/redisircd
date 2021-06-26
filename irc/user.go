package irc

import (
	"log"
	"time"

  "gopkg.in/sorcix/irc.v2"
)

type User struct {
	// Must only be written by nickServer
	Nick string
	Prefix *irc.Prefix

	// Must only be written by chanServer
	Channels map[*channel]struct{}

	client *Client
	out, err chan<- *irc.Message
}

func NewUser(c *Client) *User {
	return &User{
		Channels: make(map[*channel]struct{}),
		client: c,
	}
}

func (u *User) Send(m *irc.Message) {
	timer := time.NewTimer(10 * time.Millisecond)
	select {
	case u.out <-m:
	case <-timer.C:
		select {
		case u.err <-m:
		default:
			// dropped, but successfully signalled error anyway
		}
	}
	if !timer.Stop() {
		<-timer.C
	}
}

func (u *User) output() {
	out := make(chan *irc.Message, 512)
	u.out = out
	err := make(chan *irc.Message, 1)
	u.err = err

	go func() {
		for {
			select {
			case m := <-out:
				u.client.Encode(m)
			case <-err:
				// not keeping up, bye
				// TODO: propagate an error
				log.Printf("ok bye")
				u.client.tcpConn.Close()
				return
			}
		}
	}()
}
