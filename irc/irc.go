package irc

import (
	"log"
	"net"
	"time"

	"github.com/dgl/redisircd/ircbuf"

	"gopkg.in/sorcix/irc.v2"
)

const (
	NAME    = "redisircd"
	VERSION = "0.0.1"
)

type Server struct {
	Name      string
	RedisHost string
	Debug     bool

	cs *chanServer
	ns *nickServer
}

type Client struct {
	Server *Server
	*ircbuf.Conn
	tcpConn net.Conn

	connected      bool
	username, nick string
	last           time.Time

	User *User

	Realname string
}

func NewServer(name, redisHost string, debug bool) *Server {
	s := &Server{
		Name:      name,
		RedisHost: redisHost,
		Debug:     debug,
	}
	s.cs = NewChanServer(s)
	s.ns = NewNickServer(s)
	return s
}

func (s *Server) Listen(listen string) error {
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			if oerr, ok := err.(*net.OpError); ok {
				if oerr.Temporary() {
					log.Print(err)
					continue
				}
			}
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	c := Client{
		Server:  s,
		Conn:    ircbuf.NewConn(conn),
		tcpConn: conn,
	}

	err := c.pre()
	if err != nil {
		return
	}

	log.Printf("New connection %v", c.User.Prefix)

	c.User.output()

	err = c.commands()
	if err != nil {
		c.Encode(&irc.Message{Command: "ERROR", Params: []string{"Closing Link", err.Error()}})
	}
}
