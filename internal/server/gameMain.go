package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sethvargo/go-limiter"
	"github.com/sethvargo/go-limiter/memorystore"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }} // TODO: CHECK FOR ORIGIN
var Server *GameServer
var store limiter.Store

var (
	MaxPlayers        = 10
	MaxScore          = 2
	TimeoutMultiplier = 100
)

//go:embed data.txt
var f embed.FS

func InitServer(maxPlayers int, maxScore int, timeoutMultiplier int) {
	MaxPlayers = maxPlayers
	MaxScore = maxScore
	TimeoutMultiplier = timeoutMultiplier
	rand.Seed(time.Now().UnixNano())
	QuestionList, err := readLines("data.txt", f)
	if err != nil {
		fmt.Println(err)
		log.Fatal("Error reading questions")
	}
	store, err = memorystore.New(&memorystore.Config{
		// Number of tokens allowed per interval.
		Tokens: 30,

		// Interval until tokens reset.
		Interval: time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}
	Server = &GameServer{
		players:     make(map[*websocket.Conn]*Player),
		connections: make(map[string]*websocket.Conn),
		rooms:       make(map[string]*GameRoom),
		questions:   QuestionList,
	}
}

type Action struct {
	Action string
	Data   string
}

type ErrorMsg struct {
	MsgType   string `json:"msgType"`
	Error     string `json:"error"`
	ErrorCode int    `json:"errorCode"`
}

func wsActionHandler(c *websocket.Conn, action Action) {
	switch action.Action {
	case "register":
		Server.playerRegister(c, action.Data)
	case "login":
		Server.playerLogin(c, action.Data)
	case "joinRoom":
		player, err := Server.getPlayerByConnection(c)
		if err != nil {
			fmt.Println(err)
			return
		}
		if player.room != nil {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Player already in a room",
				ErrorCode: 22,
			})
			return
		}
		room, err := Server.getRoomById(action.Data)
		if err != nil {
			fmt.Println(err)
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Room not found",
				ErrorCode: 20,
			})
			return
		}
		if len(room.Players) == MaxPlayers {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Room is full",
				ErrorCode: 25,
			})
			return
		}
		player.joinRoom(room)
	case "createRoom":
		player, err := Server.getPlayerByConnection(c)
		if err != nil {
			fmt.Println(err)
			return
		}
		if player.room != nil {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Player already in a room",
				ErrorCode: 22,
			})
			return
		}
		player.joinRoom(Server.createRoom())
	case "leaveRoom":

		player, err := Server.getPlayerByConnection(c)
		if err != nil {
			fmt.Println("No player found", err)
			return
		}
		if player.room == nil {
			fmt.Println("Player not in room")
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Player not in a room",
				ErrorCode: 23,
			})
			return
		}
		player.leaveRoom()
	case "startGame":
		player, err := Server.getPlayerByConnection(c)
		if err != nil {
			fmt.Println(err)
			return
		}
		if player.room == nil {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Player not in room",
				ErrorCode: 21,
			})
			return
		}
		if player.room.GameStage != WaitingStage {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Game in progress",
				ErrorCode: 30,
			})
			return
		}
		if player.room.getPlayersSlice()[0] != player {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Only host is allowed to start games",
				ErrorCode: 24,
			})
			return
		}
		player.room.transitionStage()
	case "sendAnswer":
		player, err := Server.getPlayerByConnection(c)
		if err != nil {
			fmt.Println(err)
			return
		}
		if player.room == nil {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Player not in room",
				ErrorCode: 21,
			})
			return
		}
		if player.room.GameStage != WritingStage {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Not writing stage",
				ErrorCode: 31,
			})
			return
		}
		// if len(action.Data) > 50
		player.room.writingStageHandler(player, action.Data)
	case "voteAnswer":
		player, err := Server.getPlayerByConnection(c)
		if err != nil {
			fmt.Println("Player not found", err)
			return
		}
		if player.room == nil {
			fmt.Println("Player not in room")
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Player not in room",
				ErrorCode: 21,
			})
			return
		}
		if player.room.GameStage != VotingStage {
			fmt.Println("Not voting stage")
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Not voting stage",
				ErrorCode: 32,
			})
			return
		}
		player.room.votingStageHandler(player, action.Data)
	case "sendMessage":
		player, err := Server.getPlayerByConnection(c)
		if err != nil {
			fmt.Println(err)
			return
		}
		if player.room == nil {
			c.WriteJSON(&ErrorMsg{
				MsgType:   "error",
				Error:     "Player not in room",
				ErrorCode: 21,
			})
			return
		}
		if len(action.Data) == 0 {
			return
		}
		player.room.broadcastMessage(
			&struct {
				MsgType     string `json:"msgType"`
				Author      string `json:"author"`
				ChatMessage string `json:"chatMessage"`
			}{
				MsgType:     "chat",
				Author:      player.Name,
				ChatMessage: action.Data,
			})
		player.room.sendState()
	}
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func() {
		Server.playerDisconnect(c)
		c.Close()
	}()

	userAddr := strings.Split(r.RemoteAddr, ":")[0]
	if forwardedHeader := r.Header.Get("X-Forwarded-For"); len(forwardedHeader) != 0 {
		userAddr = strings.Split(forwardedHeader, ",")[0]
	}
	activityLog("conn", 2, fmt.Sprintf("Connection init with %v", userAddr))
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		_, _, _, ok, _ := store.Take(context.Background(), userAddr)
		if !ok {
			activityLog("conn", 1, fmt.Sprintf("address %v hit rate limit", userAddr))
			continue
		}
		activityLog("wsrecv", 4, string(message))
		activityLog("wsrecv", 4, fmt.Sprintf("%+v", c.RemoteAddr()))
		var action Action
		err = json.Unmarshal(message, &action)
		if err != nil {
			log.Println(err)
			break
		}
		wsActionHandler(c, action)
	}
}
