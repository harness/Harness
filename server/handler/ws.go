package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write the message to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WsHandler struct {
	pubsub  *pubsub.PubSub
	commits database.CommitManager
	perms   database.PermManager
	repos   database.RepoManager
	sess    session.Session
}

func NewWsHandler(repos database.RepoManager, commits database.CommitManager, perms database.PermManager, sess session.Session, pubsub *pubsub.PubSub) *WsHandler {
	return &WsHandler{pubsub, commits, perms, repos, sess}
}

// WsUser will upgrade the connection to a Websocket and will stream
// all events to the browser pertinent to the authenticated user. If the user
// is not authenticated, only public events are streamed.
func (h *WsHandler) WsUser(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.UserCookie(r)

	// upgrade the websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return badRequest{err}
	}

	// register a channel for global events
	channel := h.pubsub.Register("_global")
	sub := channel.Subscribe()

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		sub.Close()
		ws.Close()
	}()

	go func() {
		for {
			select {
			case msg := <-sub.Read():
				work, ok := msg.(*model.Request)
				if !ok {
					break
				}

				// user must have read access to the repository
				// in order to pass this message along
				if role := h.perms.Find(user, work.Repo); !role.Read {
					break
				}

				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteJSON(work)
				if err != nil {
					ws.Close()
					return
				}
			case <-sub.CloseNotify():
				ws.Close()
				return
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					ws.Close()
					return
				}
			}
		}
	}()

	readWebsocket(ws)
	return nil

}

// WsConsole will upgrade the connection to a Websocket and will stream
// the build output to the browser.
func (h *WsHandler) WsConsole(w http.ResponseWriter, r *http.Request) error {
	var commitID, _ = strconv.Atoi(r.FormValue(":id"))

	commit, err := h.commits.Find(int64(commitID))
	if err != nil {
		return notFound{err}
	}
	repo, err := h.repos.Find(commit.RepoID)
	if err != nil {
		return notFound{err}
	}
	user := h.sess.UserCookie(r)
	if ok, _ := h.perms.Read(user, repo); !ok {
		return notFound{err}
	}

	// find a channel that we can subscribe to
	// and listen for stream updates.
	channel := h.pubsub.Lookup(commit.ID)
	if channel == nil {
		return notFound{}
	}
	sub := channel.Subscribe()
	defer sub.Close()

	// upgrade the websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return badRequest{err}
	}

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		ws.Close()
	}()

	go func() {
		for {
			select {
			case msg := <-sub.Read():
				data, ok := msg.([]byte)
				if !ok {
					break
				}
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					log.Printf("websocket for commit %d closed. Err: %s\n", commitID, err)
					ws.Close()
					return
				}
			case <-sub.CloseNotify():
				log.Printf("websocket for commit %d closed by client\n", commitID)
				ws.Close()
				return
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					log.Printf("websocket for commit %d closed. Err: %s\n", commitID, err)
					ws.Close()
					return
				}
			}
		}
	}()

	readWebsocket(ws)
	return nil
}

// readWebsocket will block while reading the websocket data
func readWebsocket(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

// Ping is a method that is being used for internal testing and
// will be removed prior to release
func (h *WsHandler) Ping(w http.ResponseWriter, r *http.Request) error {
	channel := h.pubsub.Register("_global")
	msg := model.Request{
		Repo:   &model.Repo{ID: 1, Private: false, Host: "github.com", Owner: "drone", Name: "drone"},
		Commit: &model.Commit{ID: 1, Status: "Started", Branch: "master", Sha: "113f4917ff9174945388d86395f902cd154074cb", Message: "Remove branches by SCM hook", Author: "bradrydzewski", Gravatar: "8c58a0be77ee441bb8f8595b7f1b4e87"},
	}
	channel.Publish(&msg)
	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *WsHandler) Register(r *pat.Router) {
	r.Post("/ws/ping", errorHandler(h.Ping))
	r.Get("/ws/user", errorHandler(h.WsUser))
	r.Get("/ws/stdout/{id}", errorHandler(h.WsConsole))
}
