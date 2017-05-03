package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/pnelson/tune"
)

type broker struct {
	core    *tune.Core
	current string
	clients map[chan string]struct{}
	put     chan chan string
	del     chan chan string
}

func newBroker(core *tune.Core) *broker {
	b := &broker{
		core:    core,
		clients: make(map[chan string]struct{}),
		put:     make(chan chan string),
		del:     make(chan chan string),
	}
	go b.run()
	return b
}

func (b *broker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		abort(w, http.StatusInternalServerError)
		return
	}
	queue := make(chan string, 1)
	b.put <- queue
	n, ok := w.(http.CloseNotifier)
	if !ok {
		abort(w, http.StatusInternalServerError)
		return
	}
	go func() {
		<-n.CloseNotify()
		b.del <- queue
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	for {
		data, ok := <-queue
		if !ok {
			break
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		f.Flush()
	}
}

func (b *broker) run() {
	for {
		select {
		case queue := <-b.put:
			b.clients[queue] = struct{}{}
			if b.current != "" {
				queue <- b.current
			}
		case queue := <-b.del:
			delete(b.clients, queue)
			close(queue)
		case e := <-b.core.Events:
			m, err := json.Marshal(e)
			if err != nil {
				log.Println(err)
			}
			b.current = string(m)
			for queue := range b.clients {
				queue <- b.current
			}
		}
	}
}
