
---

# Quiz Game Server and Client

This project consists of a server and client that allow users to participate in a real-time quiz game. The server manages the game sessions, questions, and player scores. The client allows users to join a game session, receive questions, and submit their answers.

## Requirements

- Go 1.2
- Ably Realtime API Key

## Getting Started

- Questions are stored in a JSON file. You can add your own questions to the file or use the existing ones.

## Starting the Server

The server application is configured to run with specific parameters that you can set via command-line flags. Here's what each flag represents and how to use them:

- `--maxSessionCount`: Determines the maximum number of active sessions the server can manage simultaneously. Not setting this value defaults it to `2`.

- `--maxPlayers`: Sets the maximum number of players allowed in a single session. It defaults to `2` if not specified.

- `--ablyKey`: Represents your unique Ably API key,  you need to give it a real one for the server to work.

To run the server, enter the following command from the root directory of the project:
```bash
go run cmd/quiz-server/main.go --maxSessionCount=2 --maxPlayers=2 --ablyKey=your-ably-key
```
- Default port is 8080
---

### Running the Client

1. **Run the Client**

   Use the Go command to run the client:

    ```bash
    go run cmd/quiz-client/main.go --ablyKey=your-ably-key
    ```

   Follow the on-screen prompts to enter your player name and join a quiz session.

## How to Play

- Once you start the client, enter your unique player name.
- After joining a session, wait for a question to be displayed.
- Type your answer (1, 2, 3, 4 etc..) and press `Enter`.
- To leave the game, type `exit` and press `Enter`.
- Client exits when game ends

---