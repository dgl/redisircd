package irc

import (
	"net"
	"strings"

	"gopkg.in/sorcix/irc.v2"
)

// nickServer runs in a goroutine and manages nicknames
type nickServer struct {
	nicks  map[string]*User
	server *Server
	sendCh chan<- nickRequest
}

type nickReqType int

const (
	NR_NEW nickReqType = iota
	NR_CHANGE
	NR_PRIVMSG
	NR_NOTICE
	NR_QUIT
)

type nickRequest struct {
	Type   nickReqType
	Name   string
	Client *Client
	User   *User
	Params []string
	Reply  chan *User
}

func NewNickServer(server *Server) *nickServer {
	reqCh := make(chan nickRequest, 100)

	ns := &nickServer{
		nicks:  make(map[string]*User),
		server: server,
		sendCh: reqCh,
	}
	go ns.run(reqCh)
	return ns
}

func (ns *nickServer) run(reqCh <-chan nickRequest) {
	for req := range reqCh {
		switch req.Type {
		case NR_NEW:
			var user *User
			if _, ok := ns.nicks[strings.ToLower(req.Name)]; !ok {
				// Not already in use
				user = NewUser(req.Client)
				user.Nick = req.Name
				user.Prefix = &irc.Prefix{
					Name: req.Name,
					User: req.Client.username,
					Host: req.Client.tcpConn.RemoteAddr().(*net.TCPAddr).IP.String()}

				ns.nicks[strings.ToLower(req.Name)] = user
			}
			req.Reply <- user
		case NR_CHANGE:
			var user *User
			// XXX: needs to keep old prefix, then update channels, then update prefix
			/*if _, ok := ns.nicks[strings.ToLower(req.Name)]; !ok {
				oldNick := strings.ToLower(req.User.Nick)
				// Not already in use
				ns.nicks[strings.ToLower(req.Name)] = req.User
				req.User.Nick = req.Name
				delete(ns.nicks, oldNick)
			}*/
			req.Reply <- user
		case NR_PRIVMSG, NR_NOTICE:
			if user, ok := ns.nicks[strings.ToLower(req.Name)]; ok {
				cmd := "PRIVMSG"
				if req.Type == NR_NOTICE {
					cmd = "NOTICE"
				}
				user.Send(&irc.Message{
					Prefix: req.User.Prefix,
					Command: cmd,
					Params: []string{req.Name, req.Params[0]},
				})
			} else {
				/*req.User.Send(&irc.Message{
					Prefix:  &irc.Prefix{Name: ns.server.Name},
					Command: irc.ERR_NOSUCHCHANNEL, // XXX
					Params:  []string{req.User.Nick, req.Name, "No such nick"}})*/
			}
		case NR_QUIT:
			delete(ns.nicks, strings.ToLower(req.Name))
			if req.Reply != nil {
				req.Reply <- nil
			}
		}
	}
}

func (ns *nickServer) send(req nickRequest) {
	ns.sendCh <- req
}
