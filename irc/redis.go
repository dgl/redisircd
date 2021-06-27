package irc

import (
	"context"
	"log"

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
		server.cs.send(chanRequest{
			Type: CR_PRIVMSG,
			Name: channel.Name,
			// TODO: We can do better.
			User: &User{Prefix: &irc.Prefix{
				Name: name,
				User: "auto",
				Host: "redis",
			}},
			Params: []string{string(m.Message)}})
	}
}
