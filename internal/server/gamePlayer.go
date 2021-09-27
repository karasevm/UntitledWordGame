package server

import (
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/gorilla/websocket"
)

type Player struct {
	Name       string `json:"name"`
	room       *GameRoom
	Score      int  `json:"score"`
	ActionDone bool `json:"actionDone"`

	roomUpdateTimestamp int64
	disconnectTimeout   *time.Timer
	id                  string
	mu                  sync.Mutex
	connection          *websocket.Conn
}

func NewPlayer(c *websocket.Conn, name string) *Player {
	return &Player{connection: c, Name: name, id: uniuri.New(), ActionDone: false}
}

func (pl *Player) sendSelf() {
	roomName := ""
	if pl.room != nil {
		roomName = pl.room.Name
	}
	pl.connection.WriteJSON(
		&struct {
			MsgType    string `json:"msgType"`
			Name       string `json:"name"`
			Room       string `json:"room"`
			ActionDone bool   `json:"actionDone"`
		}{
			MsgType:    "self",
			Name:       pl.Name,
			Room:       roomName,
			ActionDone: pl.ActionDone,
		})
}

func (pl *Player) joinRoom(room *GameRoom) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	activityLog("player", 3, "Player", pl.Name, "attempting to join room", room.Name)
	pl.room = room
	pl.roomUpdateTimestamp = time.Now().Unix()
	room.addPlayer(pl)
	// room.Players = append(room.Players, pl)
	pl.sendSelf()
}

func (pl *Player) leaveRoom() {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.room.removePlayer(pl)
	pl.room = nil
	pl.sendSelf()
}
