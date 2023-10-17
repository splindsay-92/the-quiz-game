package quiz_server

import (
	"context"
	"errors"
	"fmt"
	"github.com/ably/ably-go/ably"
	"time"
)

type SessionManagerCommandType int

const (
	SubmitAnswer SessionManagerCommandType = iota
	JoinSession
	MoveSessionToInProgress
	EndSession
)

type SessionManagerCommand struct {
	CommandType  SessionManagerCommandType
	player       Player
	answer       int
	SessionId    string
	ResponseChan chan<- SessionManagerResponse
}

type SessionManagerResponse struct {
	Error     error
	SessionId string
}

type SessionManager struct {
	CommandChan          chan SessionManagerCommand
	waitingRooms         map[string]*Session
	inProgress           map[string]*Session
	maxPlayersPerSession int
	activeCount          int
	maxSessions          int
	ablyConnection       *ably.Realtime
	questions            []Question
}

func NewSessionManager(maxSessions int, maxPlayersPerSession int, ablyConnection *ably.Realtime, loadedQuestions []Question) *SessionManager {
	commandChan := make(chan SessionManagerCommand)
	waitingRooms := make(map[string]*Session)
	inProgress := make(map[string]*Session)
	qs := &SessionManager{
		CommandChan:          commandChan,
		waitingRooms:         waitingRooms,
		inProgress:           inProgress,
		maxSessions:          maxSessions,
		maxPlayersPerSession: maxPlayersPerSession,
		ablyConnection:       ablyConnection,
		questions:            loadedQuestions,
	}
	go qs.RunSessionManager()
	return qs
}

func (s *SessionManager) RunSessionManager() {
	fmt.Printf("Starting session manager \n")
	for cmd := range s.CommandChan {
		fmt.Printf("Handling SessionManagerCommand %v\n", cmd)
		response := s.handleCommand(cmd)
		cmd.ResponseChan <- response
	}
}

func (s *SessionManager) createSession() (string, error) {
	if s.activeCount >= s.maxSessions {
		return "", errors.New("max sessions reached, could not create a session to join")
	}
	// Logic to create a new session and its SessionManagerCommand channel
	sessionID := generateUniqueID()
	sessionConfig := SessionConfig{
		maxPlayersPerSession: s.maxPlayersPerSession,
		maxTimePerQuestion:   3 * time.Second,
		questions:            s.questions,
	}
	sessionAblyChannel := s.ablyConnection.Channels.Get(sessionID)
	ctx, cancel := context.WithCancel(context.Background())
	session := NewSession(sessionID, sessionConfig, s.CommandChan, sessionAblyChannel, cancel, ctx)
	s.waitingRooms[sessionID] = session

	s.activeCount++
	return sessionID, nil
}

func (s *SessionManager) handleCommand(cmd SessionManagerCommand) SessionManagerResponse {
	switch cmd.CommandType {
	case EndSession:
		fmt.Printf("Ending session %s\n", cmd.SessionId)
		// Logic to move a session from in-progress to waiting
		_, ok := s.inProgress[cmd.SessionId]
		if !ok {
			fmt.Printf("Session %s not found, cannot end session\n", cmd.SessionId)
		}
		s.activeCount--
		delete(s.inProgress, cmd.SessionId)
		return SessionManagerResponse{Error: nil}

	case MoveSessionToInProgress:
		// Logic to move a session from waiting to in-progress
		fmt.Printf("Moving session %s to in progress\n", cmd.SessionId)
		session, ok := s.waitingRooms[cmd.SessionId]
		if !ok {
			return SessionManagerResponse{Error: errors.New("session not found")}
		}
		s.inProgress[cmd.SessionId] = session
		delete(s.waitingRooms, cmd.SessionId)
		return SessionManagerResponse{Error: nil}

	case SubmitAnswer:
		// Logic to submit a SessionManagerCommand to a session
		fmt.Printf("Submitting answer %v to session %s\n", cmd.answer, cmd.SessionId)
		session, ok := s.inProgress[cmd.SessionId]
		if !ok {
			return SessionManagerResponse{Error: errors.New("session not found")}
		}
		return SessionManagerResponse{Error: session.SubmitAnswer(cmd.player, cmd.answer)}

	case JoinSession:
		var sessionToJoin *Session
		var sessionToJoinID string

		// Check if there are any available waiting rooms
		if len(s.waitingRooms) > 0 {
			// Get the first available waiting room
			for id, session := range s.waitingRooms {
				sessionToJoin = session
				sessionToJoinID = id
				break
			}

			// Add the player to the selected waiting room
			err := sessionToJoin.AddPlayer(cmd.player)
			return SessionManagerResponse{Error: err, SessionId: sessionToJoinID}
		}

		// If no waiting rooms are available, create a new one
		fmt.Printf("Creating a new session\n")
		sessionID, err := s.createSession()
		if err != nil {
			return SessionManagerResponse{Error: err}
		}

		// Add the player to the newly created waiting room
		err = s.waitingRooms[sessionID].AddPlayer(cmd.player)
		return SessionManagerResponse{Error: err, SessionId: sessionID}

	default:
		return SessionManagerResponse{Error: errors.New("unknown SessionManagerCommand")}
	}
}
