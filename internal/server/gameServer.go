package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
)

var signingSecret = uniuri.NewLen(256)

type GameServer struct {
	players     map[*websocket.Conn]*Player
	connections map[string]*websocket.Conn
	rooms       map[string]*GameRoom
	questions   []string
	mu          sync.Mutex
}

func (gs *GameServer) InitializeStatusBroadcaster() {
	intervalTicker := time.NewTicker(10 * time.Second)
	for {
		<-intervalTicker.C
		gs.mu.Lock()
		for _, player := range gs.players {
			player.connection.WriteJSON(
				&struct {
					MsgType     string `json:"msgType"`
					PlayerCount int    `json:"playerCount"`
					RoomCount   int    `json:"roomCount"`
				}{
					MsgType:     "status",
					PlayerCount: len(gs.players),
					RoomCount:   len(gs.rooms),
				})
		}
		gs.mu.Unlock()
	}
}

// removes empty rooms every minute
func (gs *GameServer) InitializeRoomGarbageCollector() {
	intervalTicker := time.NewTicker(1 * time.Minute)
	for {
		<-intervalTicker.C
		count := 0
		gs.mu.Lock()
		for name, room := range gs.rooms {
			if len(room.Players) == 0 {
				count++
				delete(gs.rooms, name)
			}
		}
		gs.mu.Unlock()
		if count > 0 {
			fmt.Println("Cleaned up", count, "rooms")
		}
	}
}

func (gs *GameServer) getPlayerById(id string) (*Player, error) {
	// for _, player := range gs.players {
	// 	if player.id == id {
	// 		return player, nil
	// 	}
	// }
	// return nil, fmt.Errorf("no player with id %s", id)
	var connection = gs.connections[id]
	if connection == nil {
		return nil, fmt.Errorf("no player with id %s", id)
	}
	return gs.players[gs.connections[id]], nil
}

func (gs *GameServer) getPlayerByConnection(c *websocket.Conn) (*Player, error) {
	// for _, player := range gs.players {
	// 	if player.connection == c {
	// 		return player, nil
	// 	}
	// }
	// return nil, fmt.Errorf("no player with connection %v", c)
	return gs.players[c], nil
}

func (gs *GameServer) getRoomById(roomId string) (*GameRoom, error) {
	for _, room := range gs.rooms {
		if room.Name == roomId {
			return room, nil
		}
	}
	return nil, fmt.Errorf("no room with id %v", roomId)
}

// playerRegister fires on player first connect to the server
func (gs *GameServer) playerRegister(c *websocket.Conn, name string) {
	// lock the mutex
	gs.mu.Lock()
	defer gs.mu.Unlock()
	// check if player name is unique
	// O(n) in worst case, not sure how to improve
	for _, player := range gs.players {
		if player.Name == name {
			c.WriteJSON(
				&ErrorMsg{
					MsgType:   "error",
					Error:     "Name already taken",
					ErrorCode: 11,
				})
			return
		}
	}
	// instantiate player
	newPlayer := NewPlayer(c, name)
	activityLog("conn", 3, fmt.Sprintf("PLAYER CONNECT TO SERVER: %+v", newPlayer))
	// gs.players = append(gs.players, newPlayer)
	gs.connections[newPlayer.id] = c
	gs.players[c] = newPlayer
	newPlayer.sendSelf()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": newPlayer.Name,
		"id":   newPlayer.id,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(signingSecret))
	if err != nil {
		log.Fatal(err)
	}

	c.WriteJSON(
		&struct {
			MsgType string `json:"msgType"`
			Data    string `json:"data"`
		}{
			MsgType: "jwt",
			Data:    tokenString,
		})
}

// playerLogin fires on player reconnect to the server
func (gs *GameServer) playerLogin(c *websocket.Conn, tokenString string) {
	// lock the mutex
	gs.mu.Lock()
	defer gs.mu.Unlock()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(signingSecret), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		player, err := gs.getPlayerById(fmt.Sprint(claims["id"]))
		if err == nil {
			player.mu.Lock()
			defer player.mu.Unlock()
			activityLog("conn", 3, "PLAYER REJOIN: ", player)

			player.disconnectTimeout.Stop()
			gs.players[c] = player
			delete(gs.players, player.connection)
			player.connection = c
			player.sendSelf()
			if player.room != nil {
				player.room.sendState() // TODO send state only to the rejoined player
			}
			gs.connections[player.id] = c
			return
		}
	}
	c.WriteJSON(
		&ErrorMsg{
			MsgType:   "error",
			Error:     "Invalid jwt",
			ErrorCode: 10,
		})
	fmt.Println(err)

}

// playerDisconnect fires on websocket connection disconnect
// does not mean that the player is leaving the server
func (gs *GameServer) playerDisconnect(c *websocket.Conn) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	for _, player := range gs.players {
		if player.connection == c {
			activityLog("conn", 3, fmt.Sprintf("PLAYER DISCONNECT, WAITING: %+v\n", player))
			player.mu.Lock()
			defer player.mu.Unlock()
			if player.disconnectTimeout != nil {
				player.disconnectTimeout.Reset(10 * time.Second)
				return
			}
			player.disconnectTimeout = time.NewTimer(10 * time.Second)
			go func() {
				<-player.disconnectTimeout.C
				gs.mu.Lock()
				defer gs.mu.Unlock()
				activityLog("conn", 3, fmt.Sprintf("PLAYER TIMEOUT: %+v\n", player))
				// Player should leave the room if they leave the server
				if player.room != nil {
					player.room.removePlayer(player)
				}
				// Delete player in connection list
				delete(gs.connections, player.id)
				// Delete player in player list
				delete(gs.players, c)
			}()
			return
		}
	}
}

func (gs *GameServer) createRoom() *GameRoom {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	newRoom := NewGameRoom()
	activityLog("server", 3, "Initializing new room", newRoom.Name)
	gs.rooms[newRoom.Name] = newRoom
	return newRoom
}
