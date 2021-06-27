package irc

import (
	"log"
	"net"
	"time"

	"gopkg.in/sorcix/irc.v2"
)

type Server struct {
	Name string
	RedisHost string

	cs *chanServer
	ns *nickServer
}

type Client struct {
	Server *Server
	*irc.Conn
	tcpConn net.Conn

	connected bool
	username, nick string
	last time.Time

	User *User

	Realname string
}

func NewServer(name, redisHost string) *Server {
	s := &Server{
		Name: name,
		RedisHost: redisHost,
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
		Server: s,
		Conn: irc.NewConn(conn),
	  tcpConn: conn,
	}

	err := c.pre()
	if err != nil {
		return
	}

	c.User.output()

	err = c.commands()
	if err != nil {
		c.Encode(&irc.Message{Command: "ERROR", Params: []string{"Closing Link", err.Error()}})
	}
}
