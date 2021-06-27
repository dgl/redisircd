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
	flagDebug  = flag.Bool("debug", false, "Enable debugging")
)

func main() {
	log.Println(irc.NAME, irc.VERSION, "is go!")
	flag.Parse()

	srv := irc.NewServer(*flagName, *flagRedis, *flagDebug)
	log.Fatal(srv.Listen(*flagListen))
}
