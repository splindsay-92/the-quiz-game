package quiz_server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Player struct {
	Name     string
	ID       string
	Score    int
	hasVoted bool
}

type SessionConfig struct {
	maxPlayersPerSession int
	maxTimePerQuestion   time.Duration
	questions            []Question
}

// RealtimeChannel for easier mocking tests.
type RealtimeChannel interface {
	Publish(ctx context.Context, name string, data interface{}) error
}
type Session struct {
	SessionConfig
	ID              string
	players         map[string]Player
	currentQuestion int
	quizManagerChan chan SessionManagerCommand
	mutex           sync.Mutex
	publishChannel  RealtimeChannel
	ctx             context.Context
	cancel          context.CancelFunc
}

type Answer struct {
	AnswerChoice    int `json:"answerChoice"`
	CurrentQuestion int `json:"currentQuestion"`
}

func NewSession(session string, sessionConfig SessionConfig, quizManagerChan chan SessionManagerCommand, ablyChannel RealtimeChannel, cancel context.CancelFunc, ctx context.Context) *Session {
	s := &Session{
		SessionConfig:   sessionConfig,
		quizManagerChan: quizManagerChan,
		players:         make(map[string]Player),
		ID:              session,
		publishChannel:  ablyChannel,
		ctx:             ctx,
		cancel:          cancel,
	}

	return s
}

func (s *Session) getCurrentQuestionCounter() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.currentQuestion
}

func (s *Session) getCurrentQuestion() Question {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.questions[s.currentQuestion]
}

func (s *Session) getQuestions() []Question {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.questions
}

func (s *Session) moveToNextQuestion() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.currentQuestion++
}

func (s *Session) getPlayers() map[string]Player {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.players
}
func (s *Session) setPlayer(player Player) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.players[player.ID] = player
}

func (s *Session) AddPlayer(player Player) (err error) {
	// Add a player
	fmt.Printf("Adding player %s to session %s\n", player.ID, s.ID)
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.players) < s.maxPlayersPerSession {
		s.players[player.ID] = Player{ID: player.ID, Name: player.Name, Score: 0}
	} else {
		return errors.New("Cannot add new player max players reached")
	}

	if len(s.players) == s.maxPlayersPerSession {
		// Max players reached start the session.
		go s.startQuiz()
		return nil
	}

	return nil
}

func (s *Session) publishScoreBoard() {
	// Generate and send the score board
	scoreBoard := make(map[string]int)
	for _, player := range s.getPlayers() {
		scoreBoard[player.Name] = player.Score
	}
	fmt.Printf("score board %v", scoreBoard)
	jsonData, err := json.Marshal(scoreBoard)
	if err != nil {
		fmt.Printf("Error marshalling score board: %v", err)
	}

	err = s.publishChannel.Publish(s.ctx, "quiz-update", jsonData)
	if err != nil {
		fmt.Printf("Error publishing score board: %v", err)
	}

}

func (s *Session) SubmitAnswer(player Player, answer int) error {
	player, exists := s.getPlayers()[player.ID]
	if !exists {
		return errors.New("player does not exist")
	}
	if player.hasVoted {
		return errors.New("player has already voted")
	}
	// Check if the submitted answer index is correct
	if answer == s.getCurrentQuestion().CorrectAnswer {
		// Could add a return to the user, so they know if they were correct or not
		player.Score++
	}
	player.hasVoted = true
	s.setPlayer(player)
	return nil
}

func (s *Session) publishQuestion() error {
	// Set all player hasVoted to false
	for _, player := range s.getPlayers() {
		player.hasVoted = false
		s.setPlayer(player)
	}
	// Publish the next question, send only the question and possible answers
	type QuestionPayload struct {
		Question        string   `json:"question"`
		PossibleAnswers []string `json:"possibleAnswers"`
	}
	currentQuestion := s.getCurrentQuestion()

	data := QuestionPayload{
		Question:        currentQuestion.Question,
		PossibleAnswers: currentQuestion.PossibleAnswers,
	}
	fmt.Printf("current question %v\n", currentQuestion)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = s.publishChannel.Publish(s.ctx, "new_question", jsonData)
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) startQuiz() {
	responseChan := make(chan SessionManagerResponse)
	s.quizManagerChan <- SessionManagerCommand{
		CommandType:  MoveSessionToInProgress,
		SessionId:    s.ID,
		ResponseChan: responseChan,
	}
	response := <-responseChan
	if response.Error != nil {
		fmt.Printf("error moving session to in progress: %v", response.Error)
		s.endSession()
		return
	}

	time.Sleep(500 * time.Millisecond)
	err := s.publishChannel.Publish(s.ctx, "quiz-update", fmt.Sprintf("Quiz starting in 3 seconds"))
	if err != nil {
		fmt.Printf("Error publishing quiz-starting message: %v", err)
		s.endSession()
	}

	time.Sleep(3 * time.Second)
	for {
		if s.getCurrentQuestionCounter() >= len(s.getQuestions()) {
			break
		}
		// Broadcast the current question.
		err := s.publishQuestion()
		if err != nil {
			fmt.Printf("Error publishing question: %v", err)
			s.endSession()
			return
		}
		// Wait for the duration of a question
		<-time.After(s.maxTimePerQuestion)

		// Move to the next question
		s.moveToNextQuestion()
	}
	s.publishScoreBoard()
	s.endSession()
}

func (s *Session) endSession() {
	err := s.publishChannel.Publish(s.ctx, "quiz-end", fmt.Sprintf("thank you for playing"))
	if err != nil {
		fmt.Printf("Error publishing end quiz message: %v", err)
	}
	responseChan := make(chan SessionManagerResponse)
	s.quizManagerChan <- SessionManagerCommand{
		CommandType:  EndSession,
		SessionId:    s.ID,
		ResponseChan: responseChan,
	}
	response := <-responseChan
	if response.Error != nil {
		panic(fmt.Errorf("error ending session: %v", response.Error))
	}
	s.cancel()

}
