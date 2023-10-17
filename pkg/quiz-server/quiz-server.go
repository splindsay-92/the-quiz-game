package quiz_server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ably/ably-go/ably"
	"net/http"
)

// QuizServer represents the main structure for the quiz server application.
type QuizServer struct {
	ctx                  context.Context
	commandChan          chan interface{}
	SessionManager       *SessionManager
	ServerChannel        ably.RealtimeChannel
	MaxSessionCount      int
	MaxPlayersPerSession int
}

// NewQuizServer initializes a new QuizServer instance.
func NewQuizServer(ctx context.Context, maxSessionCount, maxPlayersPerSession int, ablyPrivateKey string) (*QuizServer, error) {
	commandChan := make(chan interface{})

	ablyClient, err := ably.NewRealtime(ably.WithKey(ablyPrivateKey))
	if err != nil {
		return nil, err
	}

	loadedQuestions, err := LoadQuizQuestionsFromFile()
	if err != nil {
		return nil, err
	}

	sessionManager := NewSessionManager(maxSessionCount, maxPlayersPerSession, ablyClient, loadedQuestions)

	qs := &QuizServer{
		ctx:            ctx,
		commandChan:    commandChan,
		SessionManager: sessionManager,
	}

	fmt.Println("Quiz server started.")
	return qs, nil
}

// ConnectToSessionHandler handles the connection of a player to a session.
func (qs *QuizServer) ConnectToSessionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Processing request to connect to a session.")

	var request struct {
		PlayerName string `json:"playerName"`
		PlayerId   string `json:"playerId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Failed to parse request body.", http.StatusBadRequest)
		return
	}

	if request.PlayerName == "" || request.PlayerId == "" {
		http.Error(w, "Player name and ID are required.", http.StatusBadRequest)
		return
	}

	responseChan := make(chan SessionManagerResponse)
	qs.SessionManager.CommandChan <- SessionManagerCommand{
		CommandType: JoinSession,
		player: Player{
			Name: request.PlayerName,
			ID:   request.PlayerId,
		},
		ResponseChan: responseChan,
	}

	response := <-responseChan
	if response.Error != nil {
		http.Error(w, response.Error.Error(), http.StatusInternalServerError)
		return
	}

	var responseJSON struct {
		SessionId string `json:"sessionId"`
	}

	responseJSON.SessionId = response.SessionId
	responseJSONBytes, err := json.Marshal(responseJSON)
	if err != nil {
		http.Error(w, "Failed to marshal response.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(responseJSONBytes); err != nil {
		fmt.Printf("Error writing response: %s\n", err)
	}
}

// SubmitAnswerHandler processes the submission of a quiz answer.
func (qs *QuizServer) SubmitAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SessionId string `json:"sessionId"`
		PlayerId  string `json:"playerId"`
		Answer    int    `json:"answer"`
	}
	var responseJSON struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Failed to parse request body.", http.StatusBadRequest)
		return
	}

	if request.PlayerId == "" || request.SessionId == "" {
		http.Error(w, "Session ID and player ID are required.", http.StatusBadRequest)
		return
	}

	responseChan := make(chan SessionManagerResponse)
	qs.SessionManager.CommandChan <- SessionManagerCommand{
		CommandType: SubmitAnswer,
		player: Player{
			ID: request.PlayerId,
		},
		SessionId:    request.SessionId,
		answer:       request.Answer,
		ResponseChan: responseChan,
	}

	response := <-responseChan
	if response.Error != nil {
		http.Error(w, response.Error.Error(), http.StatusInternalServerError)
		return
	}

	responseJSON.Message = "Answer submitted successfully."
	bytes, err := json.Marshal(responseJSON)
	if err != nil {
		http.Error(w, "Failed to marshal response.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(bytes); err != nil {
		fmt.Printf("Error writing response: %s\n", err)
	}
}
