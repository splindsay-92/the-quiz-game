package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	quizServer "the-quiz-game/pkg/quiz-server"
)

func main() {
	// Define flags
	var maxSessionCount int
	var maxPlayersPerSession int
	var ablyPrivateKey string

	// Associate the flags with variables
	flag.IntVar(&maxSessionCount, "maxSessionCount", 1, "Maximum number of sessions")
	flag.IntVar(&maxPlayersPerSession, "maxPlayers", 1, "Maximum players per session")
	flag.StringVar(&ablyPrivateKey, "ablyKey", "your-default-ably-key", "Ably private key")

	// Parse the flags
	flag.Parse()

	ctx := context.Background()
	newQuiz, err := quizServer.NewQuizServer(ctx, maxSessionCount, maxPlayersPerSession, ablyPrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("starting listener\n")
	http.Handle("/connect-to-session", http.HandlerFunc(newQuiz.ConnectToSessionHandler))
	http.Handle("/submit-answer", http.HandlerFunc(newQuiz.SubmitAnswerHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))

}
