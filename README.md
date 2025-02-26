# CLI Chat App

A lightweight, terminal-based chat application built with Go that allows users to create and join chat rooms for real-time communication.

## Features

- 💬 **Real-time chat** via WebSocket connections
- 🔐 **User authentication** with secure JWT tokens
- 🚪 **Create and join rooms** with easy-to-share room codes
- 👤 **Username display** to identify participants
- 🖥️ **Terminal UI** with intuitive navigation and keyboard shortcuts
- 🌈 **Color-coded messages** for better readability
- ⌨️ **Keyboard shortcuts** for efficient navigation

## Installation

### Prerequisites

- Go 1.16 or higher
- SQLite for database storage

### From Source

Clone the repository:

```bash
git clone https://github.com/yourusername/cli-chat-app.git
cd cli-chat-app
```

Build the application:

```bash
go build -o cli-chat-app cmd/main.go
```

Run the application:

```bash
./chat-app
```

### Using Go Install

```bash
go install github.com/c0sm0thecoder/cli-chat-app@latest
```

## Usage

### Starting the server

Start with default configuration:

```bash
./chat-app server
```

Specify a custom port:

```bash
./chat-app server -port 8081
```

### Connecting as a client

Connect to a local server:

```bash
./chat-app client --server http://localhost:8080
```

Connect to a remote server:

```bash
./chat-app client --server https://your-chat-server.com
```

## Keyboard Shortcuts

- **Ctrl+Q**: Quit the application
- **ESC**: Go back to the previous screen
- **Tab**: Navigate between input fields
- **Enter**: Submit forms or send messages

## Configuration

The application can be configured through environment variables:

```bash
export PORT=8080
export JWT_SECRET=your_jwt_secret_key
export DB_PATH=./chat.db
```

Then run the server

```bash
./chat-app server
```

## Architecture

The application follows a clean architecture pattern:

- **UI Layer**: Terminal-based user interface using tview
- **Controllers**: Handle HTTP requests and route them to services
- **Services**: Implement business logic and orchestrate data flow
- **Repositories**: Manage data persistence with the database
- **Models**: Define the data structures used throughout the application
- **Realtime**: Manage WebSocket connections for real-time communication

## Technologies Used

- **Go**: Main programming language
- **tview**: Terminal UI library
- **chi**: HTTP router
- **GORM**: ORM library for SQLite
- **JWT**: Authentication tokens
- **WebSockets**: Real-time communication

## Development

### Building for Different Platforms

Build for Linux:

```bash
GOOS=linux GOARCH=amd64 go build -o chat-app-linux cmd/main.go
```

Build for Windows:

```bash
GOOS=windows GOARCH=amd64 go build -o chat-app.exe cmd/main.go
```

Build for macOS:

```
GOOS=darwin GOARCH=amd64 go build -o chat-app-mac cmd/main.go
```

## Acknowledgements

- [tview](https://github.com/rivo/tview) for the terminal UI components
- [go-chi](https://github.com/go-chi/chi) for HTTP routing
- [gorilla/websocket](https://github.com/gorilla/websocket) for WebSocket implementation
