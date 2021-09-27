package server

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/dchest/uniuri"
)

type Stage int

const (
	WaitingStage Stage = iota
	WritingStage
	VotingStage
	WinnerStage
)

type GameAnswer struct {
	Id       string `json:"id"`
	Content  string `json:"content"`
	authorId string
	votes    int
}

type GameRoom struct {
	Name         string
	GameStage    Stage
	Answers      []*GameAnswer
	Players      map[string]*Player
	Winner       *Player
	WinnerAnswer *GameAnswer
	Question     string

	t  *time.Timer
	c  chan struct{}
	mu sync.Mutex
}

type ByJoin []*Player

func (a ByJoin) Len() int           { return len(a) }
func (a ByJoin) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByJoin) Less(i, j int) bool { return a[i].roomUpdateTimestamp < a[j].roomUpdateTimestamp }

func (s *GameRoom) getPlayersSlice() []*Player {
	playerSlice := make([]*Player, 0, len(s.Players))
	fmt.Printf("Players in map\n")
	for _, pl := range s.Players {
		playerSlice = append(playerSlice, pl)
		fmt.Printf("%v", pl)
	}
	fmt.Printf("\n")
	fmt.Printf("Players in slice\n")
	sort.Sort(ByJoin(playerSlice))
	for _, pl := range playerSlice {
		fmt.Printf("%v", pl)
	}
	fmt.Printf("\n")
	return playerSlice
}

func NewGameRoom() *GameRoom {
	return &GameRoom{
		Name:      uniuri.NewLenChars(8, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")),
		GameStage: WaitingStage,
		c:         make(chan struct{}),
		Players:   make(map[string]*Player),
		Question:  "",
	}
}

func (s *GameRoom) broadcastMessage(message interface{}) {
	activityLog("room", 3, fmt.Sprintf("Broadcast in room %v: %+v %p\n", s.Name, message, &s))
	for _, player := range s.Players {
		player.connection.WriteJSON(message)
	}
}

func (s *GameRoom) addPlayer(pl *Player) {
	s.mu.Lock()
	s.Players[pl.id] = pl
	// todo: check if player joining mid game works
	s.mu.Unlock()
	s.broadcastMessage(
		&struct {
			MsgType     string `json:"msgType"`
			Author      string `json:"author"`
			ChatMessage string `json:"chatMessage"`
		}{
			MsgType:     "chat",
			Author:      "Server",
			ChatMessage: fmt.Sprintf("Player %v has joined", pl.Name),
		})
	s.sendState()
}

func (s *GameRoom) removePlayer(pl *Player) {
	// TODO: Handle player leave during game
	s.mu.Lock()
	activityLog("room", 3, "Removing player", pl.Name, "from room", s.Name)
	delete(s.Players, pl.id)
	// todo: fixup game state on player disconnect
	s.mu.Unlock()
	s.broadcastMessage(
		&struct {
			MsgType     string `json:"msgType"`
			Author      string `json:"author"`
			ChatMessage string `json:"chatMessage"`
		}{
			MsgType:     "chat",
			Author:      "Server",
			ChatMessage: fmt.Sprintf("Player %v has left", pl.Name),
		})
	s.sendState()
}

func (s *GameRoom) resetPlayerStatus() {
	fmt.Println("resetting player status in room", s.Name)
	for _, pl := range s.Players {
		pl.ActionDone = false
		pl.sendSelf()
	}
}

func (s *GameRoom) resetPlayerScore() {
	fmt.Println("resetting player score in room", s.Name)
	for _, pl := range s.Players {
		pl.Score = 0
		pl.sendSelf()
	}
}

func (s *GameRoom) sendState() {
	s.broadcastMessage(
		&struct {
			MsgType      string        `json:"msgType"`
			RoomName     string        `json:"roomName"`
			Players      []*Player     `json:"players"`
			Answers      []*GameAnswer `json:"answers"` // TODO: Randomize answer order
			GameStage    Stage         `json:"gameStage"`
			Question     string        `json:"question"`
			Winner       *Player       `json:"winner"`
			WinnerAnswer *GameAnswer   `json:"winnerAnswer"`
		}{
			MsgType:      "roomState",
			RoomName:     s.Name,
			Players:      s.getPlayersSlice(),
			Answers:      s.Answers,
			GameStage:    s.GameStage,
			Question:     s.Question,
			Winner:       s.Winner,
			WinnerAnswer: s.WinnerAnswer,
		})
}

func (s *GameRoom) transitionStage() {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.GameStage {
	case WaitingStage:
		// begin game
		if len(s.Players) < 2 {
			return
		}
		s.resetPlayerScore()
		s.resetPlayerStatus()
		s.Answers = make([]*GameAnswer, 0) // init answers
		s.GameStage = WritingStage
		s.Question = Server.questions[rand.Intn(len(Server.questions))]
		s.t = time.NewTimer((30 * time.Second * time.Duration(TimeoutMultiplier)))
		go func() {
			// wait for either the timer or the channel
			select {
			case <-s.t.C: // todo: cleanup, remove duplicate code
				s.mu.Lock()
				fmt.Println("Writing Stage timeout")

			case <-s.c:
				s.mu.Lock()
				fmt.Println("Writing Stage end")
				s.t.Stop()
			}
			s.t = nil
			s.resetPlayerStatus()
			s.mu.Unlock()
			if s.GameStage == WritingStage {
				s.transitionStage()
			}

		}()
		s.sendState()

	case WritingStage:
		// finish writing answers
		if len(s.Answers) == 0 {
			// no one answered, we should probably stop the game
			s.GameStage = WaitingStage
			s.sendState()
			return
		}
		if len(s.Answers) == 1 {
			// only one person answered, give them a technical win
			s.Players[s.Answers[0].authorId].Score++
			s.GameStage = WinnerStage
			s.sendState()
			return
		}
		s.GameStage = VotingStage
		rand.Shuffle(
			len(s.Answers),
			func(i, j int) { s.Answers[i], s.Answers[j] = s.Answers[j], s.Answers[i] },
		)
		s.t = time.NewTimer((30 * time.Second * time.Duration(TimeoutMultiplier)))
		go func() {
			// wait for either the timer or the channel
			select {
			case <-s.t.C: // todo: cleanup, remove duplicate code
				fmt.Println("Voting Stage timeout")
				s.mu.Lock()
			case <-s.c:
				fmt.Println("Voting Stage end")
				s.mu.Lock()
				s.t.Stop()
			}
			s.t = nil
			s.resetPlayerStatus()
			s.mu.Unlock()
			if s.GameStage == VotingStage {
				s.transitionStage()
			}

		}()
		s.sendState()
	case VotingStage:
		// finish voting
		bestAnswer := s.Answers[0]

		// todo: if multiple answers scored the same, the first one wins,
		// should probably do something smarted
		for _, ans := range s.Answers[1:] {
			if ans.votes > bestAnswer.votes {
				bestAnswer = ans
			}
		}
		s.WinnerAnswer = bestAnswer
		for _, pl := range s.Players {
			if bestAnswer.authorId == pl.id {
				s.Winner = pl
			}
		}
		s.GameStage = WinnerStage
		s.Winner.Score++
		fmt.Printf("Player %v won the round\n", s.Winner.Name)
		s.sendState()
		go func() {
			// tempTimer := time.NewTimer(3600 * time.Second) // winner message timeout
			// <-tempTimer.C
			s.mu.Lock()
			defer s.mu.Unlock()
			s.resetPlayerStatus()
			for _, pl := range s.Players {
				if pl.Score >= MaxScore {
					s.GameStage = WaitingStage // TODO: set stage based on winner score
					s.resetPlayerScore()
					s.sendState()
					return
				}
			}
			s.Answers = make([]*GameAnswer, 0) // init answers
			s.GameStage = WritingStage
			s.Question = Server.questions[rand.Intn(len(Server.questions))]
			s.t = time.NewTimer((30 * time.Second * time.Duration(TimeoutMultiplier)))
			go func() {
				// wait for either the timer or the channel
				select {
				case <-s.t.C: // todo: cleanup, remove duplicate code
					s.mu.Lock()
					fmt.Println("Writing Stage timeout")

				case <-s.c:
					s.mu.Lock()
					fmt.Println("Writing Stage end")
					s.t.Stop()
				}
				s.t = nil
				s.resetPlayerStatus()
				s.mu.Unlock()
				if s.GameStage == WritingStage {
					s.transitionStage()
				}

			}()
			s.sendState()
		}()
	}
}

// handle writing stage messages from players
func (s *GameRoom) writingStageHandler(author *Player, message string) {
	if len(message) < 1 {
		// message must not be empty
		return
	}
	for _, answer := range s.Answers {
		if answer.authorId == author.id {
			// player already submitted an answer
			return
		}
	}
	s.mu.Lock()
	fmt.Printf("Received writing stage message %v from player %v, current state: %+v\n", message, author.Name, s)
	defer s.mu.Unlock()
	s.Answers = append(s.Answers,
		&GameAnswer{
			authorId: author.id,
			votes:    0,
			Id:       uniuri.New(),
			Content:  message,
		})
	author.ActionDone = true
	s.sendState()
	author.sendSelf()
	if len(s.Answers) == len(s.Players) {
		fmt.Println("All players finished writing")
		s.c <- struct{}{}
	}
}

// handle voting stage messages from players
func (s *GameRoom) votingStageHandler(author *Player, answerId string) {
	// you can only vote once
	if author.ActionDone {
		fmt.Println("Player already voted")
		return
	}
	s.mu.Lock()
	fmt.Printf("Received voting stage message %v from player %v, current state: %+v\n", answerId, author.Name, s)
	for _, answer := range s.Answers {
		if answer.Id == answerId {
			answer.votes++
			author.ActionDone = true
		}
	}
	s.mu.Unlock()
	s.sendState()
	author.sendSelf()
	for _, pl := range s.Players {
		if !pl.ActionDone {
			return
		}
	}
	fmt.Println("All players finished voting")
	s.c <- struct{}{}
}
