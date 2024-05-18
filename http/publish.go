package http

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/mediocregopher/radix/v4"
)

func publishHandler(w http.ResponseWriter, r *http.Request) {
	pubsub := strings.TrimPrefix(r.URL.Path, "/publish/")

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		log.Printf("Failed to get body: %v", err)
		return
	}

	pubConn, err := radix.Dial(context.TODO(), "tcp", redisHost)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		log.Printf("Failed dial: %v", err)
		return
	}
	defer pubConn.Close()

	pubConn.Do(context.TODO(), radix.Cmd(nil, "PUBLISH", pubsub, string(body)))
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte("{}"))
}
