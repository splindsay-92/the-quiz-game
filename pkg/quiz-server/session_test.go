package quiz_server // change to your actual package name

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

// MockRealtimeChannel to make it easier to test the publishScoreBoard function.
type MockRealtimeChannel struct {
	mock.Mock
}

// Publish is a mock function.
func (m *MockRealtimeChannel) Publish(ctx context.Context, name string, data interface{}) error {
	args := m.Called(ctx, name, data)
	return args.Error(0)
}

// Basic unit test for the publishScoreBoard build the correct scoreboard function.
func TestSession_publishScoreBoard(t *testing.T) {
	// Create an instance of our test object.
	mockChannel := new(MockRealtimeChannel)

	// Create a context for the session.
	ctx := context.Background()

	// Initialize the session.
	session := NewSession("testSession", SessionConfig{ /* set necessary config */ }, nil, mockChannel, nil, ctx)

	// Simulate some players in the session.
	session.players = map[string]Player{
		"player1": {Name: "Alice", ID: "1", Score: 10, hasVoted: false},
		"player2": {Name: "Bob", ID: "2", Score: 20, hasVoted: true},
	}

	expectedScoreBoard := make(map[string]int)
	for _, player := range session.players {
		expectedScoreBoard[player.Name] = player.Score
	}

	expectedJSON, err := json.Marshal(expectedScoreBoard)
	require.NoError(t, err)

	mockChannel.On("Publish", ctx, "quiz-update", mock.MatchedBy(func(data interface{}) bool {
		// Compare the passed data with the expected JSON.
		actualJSON, ok := data.([]byte)
		if !ok {
			return false
		}
		return string(actualJSON) == string(expectedJSON)
	})).Return(nil)

	session.publishScoreBoard()

	mockChannel.AssertExpectations(t)
}
