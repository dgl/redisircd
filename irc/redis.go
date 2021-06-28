package irc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/mediocregopher/radix/v4"
	"gopkg.in/sorcix/irc.v2"
)

func redisPubsub(pubsub string, channel *channel, server *Server) chan<- *irc.Message {
	ircCh := make(chan *irc.Message)
	go redisPubsubMain(pubsub, channel, server, ircCh)
	return ircCh
}

func redisPubsubMain(pubsub string, channel *channel, server *Server, ircCh <-chan *irc.Message) {
	conn, err := radix.Dial(context.TODO(), "tcp", server.RedisHost)
	if err != nil {
		log.Printf("Failed dial: %v", err)
		return
	}
	defer conn.Close()
	pubsubClient := radix.PubSubConfig{}.New(conn)
	defer pubsubClient.Close()
	msgCh := make(chan radix.PubSubMessage)

	pubConn, err := radix.Dial(context.TODO(), "tcp", server.RedisHost)
	if err != nil {
		log.Printf("Failed dial: %v", err)
		return
	}
	defer pubConn.Close()

	name := pubsub
	err = pubsubClient.Subscribe(context.TODO(), msgCh, name)
	if err != nil {
		log.Printf("Failed subscribe: %v", err)
		return
	}

	log.Printf("Subscribed to %v", name)

	for {
		select {
		case m := <-msgCh:
			text := string(m.Message)
			name := name

			if channel.redisType == "json" {
				var j interface{}
				err := json.Unmarshal(m.Message, &j)
				if err == nil {
					if len(channel.redisTextPath) > 0 {
						if res, err := jsonpath.Get(channel.redisTextPath, j); err != nil {
							text = fmt.Sprintf("%q [%v]", string(m.Message), err)
						} else {
							text = fmt.Sprintf("%v", res)
						}
					}
					if len(channel.redisNickPath) > 0 {
						if res, err := jsonpath.Get(channel.redisNickPath, j); err != nil {
							name = "redis"
							text = fmt.Sprintf("%q [%v]", string(m.Message), err)
						} else if s, ok := res.(string); ok {
							// Need an actual string, also make sure there's no spaces, as
							// that totally breaks the IRC protocol...
							name = strings.Split(strings.Split(s, "\n")[0], " ")[0]

						}
					}
				} else {
					text = fmt.Sprintf("%q [%v]", string(m.Message), err)
				}
			}

			for _, line := range strings.Split(text, "\n") {
				if len(line) == 0 {
					continue
				}
				server.cs.send(chanRequest{
					Type: CR_PRIVMSG,
					Name: channel.Name,
					// TODO: We can do better.
					User: &User{Prefix: &irc.Prefix{
						Name: name,
						User: "auto",
						Host: "redis",
					}},
					Params: []string{line}})
			}

		case m := <-ircCh:
			if m == nil {
				return
			}
			// Avoid loops, even if they should be unlikely given we force a
			// different output channel.
			if m.Prefix.Host != "redis" {
				pubConn.Do(context.TODO(), radix.Cmd(nil, "PUBLISH", pubsub+":out", m.Prefix.Name+" "+m.Params[1]))
			}
		}
	}
}
