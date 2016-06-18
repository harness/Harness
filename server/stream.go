package server

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/drone/drone/bus"
	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/store"
	"github.com/drone/drone/stream"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/manucorporat/sse"
)

// GetRepoEvents will upgrade the connection to a Websocket and will stream
// event updates to the browser.
func GetRepoEvents(c *gin.Context) {
	repo := session.Repo(c)
	c.Writer.Header().Set("Content-Type", "text/event-stream")

	eventc := make(chan *bus.Event, 1)
	bus.Subscribe(c, eventc)
	defer func() {
		bus.Unsubscribe(c, eventc)
		close(eventc)
		logrus.Infof("closed event stream")
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-eventc:
			if event == nil {
				logrus.Infof("nil event received")
				return false
			}

			// TODO(bradrydzewski) This is a super hacky workaround until we improve
			// the actual bus. Having a per-call database event is just plain stupid.
			if event.Repo.FullName == repo.FullName {

				var payload = struct {
					model.Build
					Jobs []*model.Job `json:"jobs"`
				}{}
				payload.Build = event.Build
				payload.Jobs, _ = store.GetJobList(c, &event.Build)
				data, _ := json.Marshal(&payload)

				sse.Encode(w, sse.Event{
					Event: "message",
					Data:  string(data),
				})
			}
		case <-c.Writer.CloseNotify():
			return false
		}
		return true
	})
}

func GetStream(c *gin.Context) {

	repo := session.Repo(c)
	buildn, _ := strconv.Atoi(c.Param("build"))
	jobn, _ := strconv.Atoi(c.Param("number"))

	c.Writer.Header().Set("Content-Type", "text/event-stream")

	build, err := store.GetBuildNumber(c, repo, buildn)
	if err != nil {
		logrus.Debugln("stream cannot get build number.", err)
		c.AbortWithError(404, err)
		return
	}
	job, err := store.GetJobNumber(c, build, jobn)
	if err != nil {
		logrus.Debugln("stream cannot get job number.", err)
		c.AbortWithError(404, err)
		return
	}

	rc, err := stream.Reader(c, stream.ToKey(job.ID))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	go func() {
		<-c.Writer.CloseNotify()
		rc.Close()
	}()

	var line int
	var scanner = bufio.NewScanner(rc)
	for scanner.Scan() {
		line++
		var err = sse.Encode(c.Writer, sse.Event{
			Id:    strconv.Itoa(line),
			Event: "message",
			Data:  scanner.Text(),
		})
		if err != nil {
			break
		}
		c.Writer.Flush()
	}

	logrus.Debugf("Closed stream %s#%d", repo.FullName, build.Number)
}

var (
	// Time allowed to write the file to the client.
	writeWait = 5 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Poll file for changes with this period.
	filePeriod = 10 * time.Second
)

// EventStream produces the User event stream, sending all repository, build
// and agent events to the client.
func EventStream(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logrus.Errorf("Cannot upgrade websocket. %s", err)
		}
		return
	}

	user := session.User(c)
	repo := map[string]bool{}
	if user != nil {
		repo, _ = cache.GetRepoMap(c, user)
	}

	quitc := make(chan bool)
	eventc := make(chan *bus.Event, 10)
	bus.Subscribe(c, eventc)
	defer func() {
		bus.Unsubscribe(c, eventc)
		quitc <- true
		close(quitc)
		close(eventc)
		ws.Close()
		logrus.Debug("Successfully closed websocket")
	}()

	go func() {
		for {
			select {
			case <-quitc:
			case event := <-eventc:
				if event == nil {
					return
				}
				if repo[event.Repo.FullName] || !event.Repo.IsPrivate {
					ws.WriteJSON(event)
				}
			}
		}
	}()

	reader(ws)
}

func reader(ws *websocket.Conn) {
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
