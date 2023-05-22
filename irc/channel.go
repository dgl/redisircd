package irc

import (
	"log"
	"strings"

	"gopkg.in/sorcix/irc.v2"
)

// chanServer runs in a goroutine and manages channels
type chanServer struct {
	channels map[string]*channel
	server   *Server
	sendCh   chan<- chanRequest
}

type channel struct {
	Name       string
	Users      map[*User]struct{}
	SimpleMode chanModes

	redisPubsub                             chan<- *irc.Message
	redisType, redisTextPath, redisNickPath string
	redisPublish                            bool
}

type chanModes int

const (
	CM_NONE chanModes = iota << 1
	CM_NOEXT
)

type chanReqType int

const (
	CR_JOIN chanReqType = iota
	CR_PRIVMSG
	CR_NOTICE
	CR_MODE
	CR_LEAVE
	CR_QUIT
)

type chanRequest struct {
	Type   chanReqType
	Name   string
	User   *User
	Params []string
}

func NewChanServer(server *Server) *chanServer {
	reqCh := make(chan chanRequest, 100)

	cs := &chanServer{
		channels: make(map[string]*channel),
		server:   server,
		sendCh:   reqCh,
	}
	go cs.run(reqCh)
	return cs
}

func (cs *chanServer) run(reqCh <-chan chanRequest) {
	for req := range reqCh {
		ch, chOk := cs.channels[strings.ToLower(req.Name)]

		switch req.Type {
		case CR_JOIN:
			if !chOk {
				// Need to create it
				ch = &channel{
					Name:  req.Name,
					Users: make(map[*User]struct{}),
				}
				cs.channels[strings.ToLower(req.Name)] = ch
			}
			if _, ok := ch.Users[req.User]; !ok {
				// User not already in channel
				ch.join(req.User, cs.server)
			}
			if !chOk && len(req.Name) > 1 {
				ch.mode(nil, []string{"+RP", strings.ToLower(req.Name)[1:]}, cs.server)
			}
		case CR_QUIT:
			// Have to handle quit differently to leave, as it's not channel specific.
			cs.quit(req.User, req.Params)

		case CR_PRIVMSG, CR_NOTICE:
			if chOk {
				ch.msg(req.Type, req.User, req.Params)
			} else {
				req.User.Send(&irc.Message{
					Prefix:  &irc.Prefix{Name: cs.server.Name},
					Command: irc.ERR_NOSUCHCHANNEL,
					Params:  []string{req.User.Nick, req.Name, "No such channel"}})
			}
		case CR_MODE:
			if chOk {
				if req.Params == nil {
					ch.modeSend(req.User, cs.server)
				} else {
					ch.mode(req.User, req.Params, cs.server)
				}
			} else {
				req.User.Send(&irc.Message{
					Prefix:  &irc.Prefix{Name: cs.server.Name},
					Command: irc.ERR_NOSUCHCHANNEL,
					Params:  []string{req.User.Nick, req.Name, "No such channel"}})
			}
		case CR_LEAVE:
			if chOk {
				ch.leave(req.User, req.Params, cs.server)
				if len(ch.Users) == 0 {
					delete(cs.channels, strings.ToLower(req.Name))
				}
			} else {
				req.User.Send(&irc.Message{
					Prefix:  &irc.Prefix{Name: cs.server.Name},
					Command: irc.ERR_NOSUCHCHANNEL,
					Params:  []string{req.User.Nick, req.Name, "No such channel"}})
			}
		}
	}
}

func (cs *chanServer) send(req chanRequest) {
	cs.sendCh <- req
}

func (cs *chanServer) quit(user *User, params []string) {
	um := map[*User]struct{}{}
	for ch := range user.Channels {
		for u := range ch.Users {
			if u != user {
				um[u] = struct{}{}
			}
		}
		delete(ch.Users, user)
		if len(ch.Users) == 0 {
			delete(cs.channels, strings.ToLower(ch.Name))
		}
	}
	msg := &irc.Message{
		Prefix:  user.Prefix,
		Command: "QUIT",
		Params:  []string{params[0]},
	}
	for u := range um {
		u.Send(msg)
	}
	user.Channels = nil
}

func (ch *channel) join(user *User, server *Server) {
	ch.Users[user] = struct{}{}
	user.Channels[ch] = struct{}{}

	msg := &irc.Message{
		Prefix:  user.Prefix,
		Command: "JOIN",
		Params:  []string{ch.Name},
	}
	for u := range ch.Users {
		u.Send(msg)
	}

	sp := &irc.Prefix{Name: server.Name}
	// TODO: split names into multiple lines if needed
	var sb strings.Builder
	i := 0
	for u := range ch.Users {
		if i > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString(u.Nick)
		i++
	}
	user.Send(&irc.Message{
		Prefix:  sp,
		Command: irc.RPL_NAMREPLY,
		Params:  []string{user.Nick, "=", ch.Name, sb.String()}})
	user.Send(&irc.Message{
		Prefix:  sp,
		Command: irc.RPL_ENDOFNAMES,
		Params:  []string{user.Nick, ch.Name, "End of NAMES list"}})
}

func (ch *channel) leave(user *User, params []string, server *Server) {
	if _, ok := ch.Users[user]; !ok {
		user.Send(&irc.Message{
			Prefix:  &irc.Prefix{Name: server.Name},
			Command: irc.ERR_NOTONCHANNEL,
			Params:  []string{user.Nick, ch.Name, "You're not on that channel"}})
		return
	}

	delete(ch.Users, user)
	delete(user.Channels, ch)

	msg := &irc.Message{
		Prefix:  user.Prefix,
		Command: "PART",
		Params:  []string{ch.Name, params[0]},
	}
	for u := range ch.Users {
		u.Send(msg)
	}
}

func (ch *channel) msg(t chanReqType, user *User, params []string) {
	cmd := "PRIVMSG"
	if t == CR_NOTICE {
		cmd = "NOTICE"
	}

	msg := &irc.Message{
		Prefix:  user.Prefix,
		Command: cmd,
		Params:  []string{ch.Name, params[0]},
	}

	if cmd == "PRIVMSG" && ch.redisPublish && ch.redisPubsub != nil {
		ch.redisPubsub <- msg
	}

	for u := range ch.Users {
		if u != user {
			u.Send(msg)
		}
	}
}

func (ch *channel) modeSend(user *User, server *Server) {
	mode := "+"
	if ch.SimpleMode&CM_NOEXT == CM_NOEXT {
		mode += "n"
	}
	if ch.redisType == "json" {
		mode += "J"
	}
	if ch.redisPubsub != nil {
		mode += "R"
	}
	if ch.redisNickPath != "" {
		mode += "N"
	}
	if ch.redisTextPath != "" {
		mode += "T"
	}

	user.Send(&irc.Message{
		Prefix:  &irc.Prefix{Name: server.Name},
		Command: irc.RPL_CHANNELMODEIS,
		Params:  []string{user.Nick, ch.Name, mode}})
}

func (ch *channel) mode(user *User, params []string, server *Server) {
	if len(params) < 1 {
		return
	}

	modes := params[0]
	paramIdx := 1

	state := '+'
	bad := ' '
	var modeChange strings.Builder
	var modeParam []string
	for _, c := range modes {
		switch c {
		case '+', '-':
			state = c
		case 'n':
			old := ch.SimpleMode
			if state == '+' {
				ch.SimpleMode |= CM_NOEXT
			} else {
				ch.SimpleMode &= ^CM_NOEXT
			}
			if ch.SimpleMode != old {
				modeChange.WriteRune(state)
				modeChange.WriteRune(c)
			}
		case 'b':
			// Just ignore for now, stops errors in Irssi

		// The Redis specific modes...
		case 'R':
			if ch.redisPubsub != nil {
				close(ch.redisPubsub)
				ch.redisPubsub = nil
				if state == '-' {
					modeChange.WriteRune(state)
					modeChange.WriteRune(c)
				}
			}

			if state == '+' && len(params) > paramIdx {
				p := params[paramIdx]
				paramIdx++
				modeParam = append(modeParam, p)
				modeChange.WriteRune(state)
				modeChange.WriteRune(c)

				ch.redisPubsub = redisPubsub(p, ch, server)
			}
		case 'J':
			if state == '+' {
				ch.redisType = "json"
			} else {
				ch.redisType = "plain"
			}
			modeChange.WriteRune(state)
			modeChange.WriteRune(c)

		case 'N':
			if len(params) > paramIdx {
				p := params[paramIdx]
				paramIdx++
				modeChange.WriteRune(state)
				modeChange.WriteRune(c)
				modeParam = append(modeParam, p)

				if state == '+' {
					ch.redisNickPath = p
				} else {
					ch.redisNickPath = ""
				}
			}

		case 'T':
			if len(params) > paramIdx {
				p := params[paramIdx]
				paramIdx++
				modeChange.WriteRune(state)
				modeChange.WriteRune(c)
				modeParam = append(modeParam, p)

				if state == '+' {
					ch.redisTextPath = p
				} else {
					ch.redisTextPath = ""
				}
			}

		case 'P':
			ch.redisPublish = state == '+'
			modeChange.WriteRune(state)
			modeChange.WriteRune(c)

		default:
			bad = c
			break
		}
	}

	mc := modeChange.String()
	// TODO: compress -/+ states
	if len(mc) > 0 {
		p := irc.ParsePrefix(server.Name)
		if user != nil {
			p = user.Prefix
		}
		msg := &irc.Message{
			Prefix:  p,
			Command: "MODE",
			Params:  append([]string{ch.Name, mc}, modeParam...)}
		log.Print(msg)
		for u := range ch.Users {
			u.Send(msg)
		}
	}

	if bad != ' ' {
		user.Send(&irc.Message{
			Prefix:  &irc.Prefix{Name: server.Name},
			Command: irc.ERR_UNKNOWNMODE,
			Params:  []string{user.Nick, string(bad), "is an unknown mode character"}})
		return
	}
}
