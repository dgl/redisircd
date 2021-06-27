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

func redisPubsub(channel *channel, server *Server) {
	conn, err := radix.Dial(context.TODO(), "tcp", server.RedisHost)
	if err != nil {
		log.Printf("Failed dial: %v", err)
		return
	}
	pubsubClient := radix.PubSubConfig{}.New(conn)
	msgCh := make(chan radix.PubSubMessage)

	name := channel.redisPubsub
	err = pubsubClient.Subscribe(context.TODO(), msgCh, name)
	if err != nil {
		log.Printf("Failed subscribe: %v", err)
		return
	}

	log.Printf("Subscribed to %v", name)

	for m := range msgCh {
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
						text = fmt.Sprintf("%s", res)
					}
				}
				if len(channel.redisNickPath) > 0 {
					if res, err := jsonpath.Get(channel.redisNickPath, j); err != nil {
						name = "redis"
						text = fmt.Sprintf("%q [%v]", string(m.Message), err)
					} else if s, ok := res.(string); ok {
						// Need an actual string, also make sure there's no spaces, as
						// that totally breaks the IRC protocol...
						name = strings.Split(s, " ")[0]
					}
				}
			} else {
				text = fmt.Sprintf("%q [%v]", string(m.Message), err)
			}
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
			Params: []string{text}})
	}
}
