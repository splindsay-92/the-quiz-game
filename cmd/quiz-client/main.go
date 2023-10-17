package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ably/ably-go/ably"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Client holds the ablyClient and servers url
type Client struct {
	ablyClient *ably.Realtime
	serverURL  string
}

// NewClient initializes a new Client. It establishes a connection to the Ably service
// and returns an instance of Client or an error.
func NewClient(serverURL, ablyKey string) (*Client, error) {
	realtime, err := ably.NewRealtime(ably.WithKey(ablyKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Ably realtime client: %w", err)
	}
	return &Client{
		ablyClient: realtime,
		serverURL:  serverURL,
	}, nil
}

// ConnectToSession sends a request to join a gaming session. It accepts the player's name
// and ID, and if successful, returns the session ID of the new session.
func (c *Client) ConnectToSession(playerName, playerId string) (string, error) {
	data := map[string]string{
		"playerName": playerName,
		"playerId":   playerId,
	}
	body, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Send a request to the server to join a session.
	resp, err := http.Post(c.serverURL+"/connect-to-session", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Decode the response to retrieve the session ID.
	var response struct {
		SessionId string `json:"sessionId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Return the session ID received from the server.
	return response.SessionId, nil
}

// QuestionMessage represents the structure of a message containing a question and answers.
type QuestionMessage struct {
	Question string   `json:"question"`
	Answers  []string `json:"possibleAnswers"`
}

// ListenToAblyChannel subscribes to a specific Ably channel, and listens for messages.
// new_question, quiz-update, and quiz-end messages are handled.
func (c *Client) ListenToAblyChannel(ctx context.Context, channelName string, cancel context.CancelFunc) {
	channel := c.ablyClient.Channels.Get(channelName)
	// Subscribe to messages on the channel.
	_, err := channel.SubscribeAll(ctx, func(msg *ably.Message) {
		// Handle the message based on its name.
		switch msg.Name {
		case "new_question":
			var questionMsg QuestionMessage
			jsonData, ok := msg.Data.(string) // Asserting that Data is a string.
			if !ok {
				fmt.Println("Error: expected string data in message")
				return
			}
			err := json.Unmarshal([]byte(jsonData), &questionMsg)
			if err != nil {
				fmt.Printf("Error unmarshalling JSON: %s\n", err)
				return
			}
			// Display the question and answers.
			c.displayQuestionAndAnswers(questionMsg)

		case "quiz-update":
			// Further actions can be taken here based on quiz updates.
			fmt.Println(msg.Data)

		case "quiz-end":
			fmt.Println("Quiz has ended.")
			// This informs the parent function that it can terminate or clean up as needed.
			cancel()
			return

		}
	})
	if err != nil {
		fmt.Printf("Error subscribing to channel: %v\n", err)
		return
	}

}

// SubmitAnswer sends the player's answer to the server.
func (c *Client) SubmitAnswer(sessionId, playerId, answer string) error {
	// Convert the answer to an integer.
	answerInt, err := strconv.Atoi(answer)
	if err != nil {
		return fmt.Errorf("error converting answer to integer: %w", err)
	}
	if answerInt < 1 {
		return fmt.Errorf("answer must be value of 1 or higher")
	}
	answerInt-- // Subtract 1 to convert to zero-based index.

	requestPayload := struct {
		SessionId string `json:"sessionId"`
		PlayerId  string `json:"playerId"`
		Answer    int    `json:"answer"`
	}{
		SessionId: sessionId,
		PlayerId:  playerId,
		Answer:    answerInt,
	}
	body, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	// Send the answer to the server.
	resp, err := http.Post(c.serverURL+"/submit-answer", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-OK responses.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}
		return fmt.Errorf("error submitting answer: %s", string(bodyBytes))
	}

	return nil
}

// displayQuestionAndAnswers outputs the question and possible answers to the console.
func (c *Client) displayQuestionAndAnswers(qm QuestionMessage) {
	fmt.Println("New question: ", qm.Question)
	fmt.Println("Select an answer from the following options:")
	for i, answer := range qm.Answers {
		fmt.Printf("%d: %s\n", i+1, answer)
	}
}

func getUserInput(reader *bufio.Reader) (string, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func setupClient(serverURL, ablyKey string) (*Client, error) {
	client, err := NewClient(serverURL, ablyKey)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}
	return client, nil
}

func joinSession(client *Client, playerName, playerId string) (string, error) {
	sessionId, err := client.ConnectToSession(playerName, playerId)
	if err != nil {
		return "", fmt.Errorf("error connecting to session: %w", err)
	}
	return sessionId, nil
}

func monitorSessionEnd(ctx context.Context) {
	select {
	case <-ctx.Done():
		fmt.Println("Session has ended.")
		os.Exit(0)
	}
}

func main() {
	serverURL := "http://localhost:8080" // Change to your server's URL.
	var ablyPrivateKey string
	// Associate the flags with variables
	flag.StringVar(&ablyPrivateKey, "ablyKey", "your-default-ably-key", "Ably private key")
	// Parse the flags
	flag.Parse()

	playerId := uuid.New().String() // Unique player ID generated here.                                    // Unique player ID generated here.
	reader := bufio.NewReader(os.Stdin)

	// Set up the client.
	client, err := setupClient(serverURL, ablyPrivateKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get the player's name and join the session.
	fmt.Println("Enter your name:")
	playerName, err := getUserInput(reader)
	if err != nil {
		fmt.Println("Error reading player name:", err)
		return
	}

	sessionId, err := joinSession(client, playerName, playerId)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	fmt.Printf("Connected to session, please wait for the session to start %s\n", sessionId)
	fmt.Println("During the game, enter answers in the following format: 1, 2, 3, 4, 5... or type 'exit' to leave.")

	go monitorSessionEnd(ctx)
	go client.ListenToAblyChannel(ctx, sessionId, cancel)

	// Main loop for player input.
	for {
		answer, err := getUserInput(reader)
		if err != nil {
			fmt.Println("Error reading answer:", err)
			continue
		}

		if answer == "exit" {
			break
		}

		err = client.SubmitAnswer(sessionId, playerId, answer)
		if err != nil {
			fmt.Println("Error submitting answer:", err)
			continue
		}

		fmt.Printf("Submitted answer %s\n", answer)
	}
}
