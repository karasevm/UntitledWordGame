package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"

	"fgame/internal/server"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var maxPlayers = flag.Int("maxPlayers", 10, "maximum number of players in room")
var maxScore = flag.Int("maxScore", 10, "maximum score for player")
var timeoutMultiplier = flag.Int("timeoutMultiplier", 1, "maximum score for player")

//go:embed web
var webFS embed.FS

func handleSPA(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = "/web" + r.URL.Path
	http.FileServer(http.FS(webFS)).ServeHTTP(w, r)
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	fmt.Printf("Initializing server on address %v with maxPlayers = %v, maxScore = %v, timeoutMultiplier = %v\n", *addr, *maxPlayers, *maxScore, *timeoutMultiplier)
	server.InitServer(*maxPlayers, *maxScore, *timeoutMultiplier)
	http.HandleFunc("/ws", server.WsHandler)
	http.HandleFunc("/", handleSPA)
	go server.Server.InitializeRoomGarbageCollector()
	go server.Server.InitializeStatusBroadcaster()
	log.Default().Println("Starting server on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
