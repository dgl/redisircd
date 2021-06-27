package main

import (
	"flag"
	"log"
	"os"

	"github.com/dgl/redisircd/irc"
)

var (
	flagListen = flag.String("listen", "localhost:6667", "[ip]:port to listen on for IRC connections")
	flagName   = flag.String("name", func() string { h, _ := os.Hostname(); return h }(), "Hostname of the server")
	flagRedis  = flag.String("redis", "localhost:6379", "host:port to connect to Redis at")
)

func main() {
	flag.Parse()

	srv := irc.NewServer(*flagName, *flagRedis)
	log.Fatal(srv.Listen(*flagListen))
}
