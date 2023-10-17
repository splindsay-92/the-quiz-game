package quiz_server

import (
	"encoding/json"
	"os"
)

type Question struct {
	Question        string   `json:"question"`
	PossibleAnswers []string `json:"possibleAnswers"`
	CorrectAnswer   int      `json:"correctAnswer"`
}

func LoadQuizQuestionsFromFile() ([]Question, error) {
	var questions []Question
	fileContent, err := os.ReadFile("resources/questions.json")
	if err != nil {
		return questions, err
	}

	err = json.Unmarshal(fileContent, &questions)
	if err != nil {
		return questions, err
	}
	return questions, nil

}
