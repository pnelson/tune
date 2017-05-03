package http

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/pnelson/tune"
)

// mux is a HTTP multiplexer.
type mux struct {
	core   *tune.Core
	broker *broker
	public http.Handler
}

// Serve listens on the configured TCP address and dispatches
// incoming HTTP requests to the application request multiplexer.
func Serve(core *tune.Core) error {
	m := &mux{
		core:   core,
		broker: newBroker(core),
		public: http.FileServer(http.Dir(core.Config.PublicDir)),
	}
	return http.ListenAndServe(core.Config.Addr, m)
}

func (m *mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case strings.HasPrefix(req.URL.Path, "/channels.json"):
		m.channels(w, req)
	case strings.HasPrefix(req.URL.Path, "/play/"):
		m.play(w, req)
	case req.URL.Path == "/stop":
		m.stop(w, req)
	case req.URL.Path == "/events":
		m.broker.ServeHTTP(w, req)
	default:
		m.public.ServeHTTP(w, req)
	}
}

type channel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (m *mux) channels(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		abort(w, http.StatusMethodNotAllowed)
		return
	}
	channels := make(map[string][]channel)
	for s, cs := range tune.Channels {
		channels[s] = make([]channel, 0)
		for id, c := range cs {
			channels[s] = append(channels[s], channel{ID: id, Name: c.Name})
		}
		sort.Slice(channels[s], func(i, j int) bool {
			return channels[s][i].Name < channels[s][j].Name
		})
	}
	b, err := json.Marshal(channels)
	if err != nil {
		resolve(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err = w.Write(b)
	if err != nil {
		log.Println(err)
	}
}

func (m *mux) play(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		abort(w, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(req.URL.Path, "/", 4)
	if len(parts) != 4 {
		abort(w, http.StatusNotFound)
		return
	}
	station, channel := parts[2], parts[3]
	id, err := strconv.Atoi(channel)
	if err != nil {
		resolve(w, err)
		return
	}
	err = m.core.Play(station, id)
	if err != nil {
		resolve(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (m *mux) stop(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		abort(w, http.StatusMethodNotAllowed)
		return
	}
	err := m.core.Stop()
	if err != nil {
		resolve(w, err)
		return
	}
	m.core.Events <- tune.Event{}
	w.WriteHeader(http.StatusOK)
}

func abort(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

func resolve(w http.ResponseWriter, err error) {
	switch err {
	case tune.ErrNotFound:
		abort(w, http.StatusNotFound)
	default:
		log.Println(err)
		abort(w, http.StatusInternalServerError)
	}
}
