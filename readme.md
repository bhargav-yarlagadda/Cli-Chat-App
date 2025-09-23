CLI Chat App — End‑to‑End Encrypted, Connection‑Gated Chat (Go)

Overview

This project is a two-part Go application:
- Server (Fiber + WebSockets + Postgres + GORM): authentication, connection requests, message relay, persistence.
- Client (CLI): register/login, manage connection requests, and chat over WebSockets. Messages are end‑to‑end encrypted using RSA; the server never sees plaintext.

Key Features

- End‑to‑end encryption with RSA (PKCS#1 v1.5):
  - Each user generates a 2048‑bit key pair during registration.
  - Client stores the private key locally in `keys/<username>_private.pem` and uploads only the public key to the server.
  - Messages are encrypted on the sender’s client with the receiver’s public key and decrypted on the receiver’s client with their private key.
  - The server stores only ciphertext in the database.
- JWT authentication for API and WebSocket access.
- Connection gating: users must accept a connection request before they can chat.
- Pending message delivery: undelivered messages are sent when a user reconnects.

Architecture

- Server
  - Framework: Fiber (`server/main.go`).
  - Auth: `server/handlers/auth.go` (bcrypt password hashing, JWT issuance, user lookup).
  - Connections: `server/handlers/connections.go` (send/respond/list/pending counts).
  - WebSocket chat: `server/handlers/websockets.go` (JWT‑gated, validates connection, relays encrypted payloads; stores in `db.Message`).
  - Middleware: `server/middleware/Auth.go` and `server/middleware/Connections.go` (JWT parse/verify, set `Locals`).
  - DB: `server/db/db.go`, models in `server/db/schema.go` (`User`, `Connection`, `Message`). Auto‑migrates.
- Client
  - Entry: `client/main.go` with interactive prompt (`go-prompt`).
  - Commands: `client/commands/*.go` (`register`, `login`, `add`, `view-requests`, `respond`, `chat`).
  - HTTP utilities: `client/utils/utils.go` (REST base URL, user lookups).
  - WebSocket utilities: `client/utils/message.go` (JWT header dial, send/receive loop).

Security & Encryption Details

- Passwords: hashed with bcrypt on the server at registration (`bcrypt.GenerateFromPassword`).
- JWTs: HMAC‑SHA256 signed using `JWT_SECRET` from environment; required for all protected REST endpoints and WebSocket connections.
- E2E Message Encryption (client‑side in `client/commands/chat.go`):
  - Receiver public key is fetched via `GET /auth/user-info?username=<username>`.
  - Sender encrypts plaintext with the receiver’s RSA public key. For messages longer than the RSA block size, the client chunks the message into <=245‑byte segments, encrypts each, and prefixes each chunk with a 2‑byte length header. Ciphertext is Base64‑encoded for transport.
  - Receiver decodes Base64 and decrypts chunks with their RSA private key, reassembling the plaintext.
- Server never decrypts content; it validates connections and stores ciphertext in `messages.content`.

Environment Variables

Create a `.env` in both `server/` and `client/` roots as needed.

- Server `.env` (required):
  - `PORT` — e.g., `:8080`
  - `DB_URL` — Postgres DSN, e.g., `postgres://user:password@localhost:5432/chatapp?sslmode=disable`
  - `JWT_SECRET` — strong random string for HMAC signing

- Client `.env` (optional quality‑of‑life):
  - `JWT_TOKEN` — set automatically after `login`; you can pre‑seed for testing
  - `CURRENT_USER` — set automatically after `login`

Note: Client uses `client/utils/utils.go: BaseURL = "http://localhost:8080"` and `ws://localhost:8080/chat` in `chat.go`. Adjust these if your server runs elsewhere.

Install & Run

Prerequisites:
- Go 1.23+
- PostgreSQL instance reachable via `DB_URL`

1) Start the Server

```
cd server
go mod download
go run .
```

Server routes:
- `POST /auth/register` — body: `{ username, password, public_key }`
- `POST /auth/login` — body: `{ username, password }` → `{ token }`
- `GET /auth/user-info?username=<name>` — returns `{ user: { id, username, public_key, created_at } }`
- `GET /connections/pending/count` — requires `Authorization: Bearer <token>`
- `GET /connections/pending` — list pending requests (receiver)
- `POST /connections/connect` — body: `{ username }` to send request
- `POST /connections/respond` — body: `{ request_id, action: "accept"|"reject" }`
- `GET /chat` — WebSocket endpoint (JWT in `Authorization` header)

2) Start the Client

```
cd client
go mod download
go run .
```

You’ll see an interactive prompt `>`. Use `help` for available commands.

Client Commands

- register — create a new account and generate RSA keys
  - Usage: `register --username:<name> --password:<pass>`
  - If flags omitted, you’ll be prompted. A private key is saved to `keys/<username>_private.pem`.

- login — authenticate and store JWT in session env
  - Usage: `login --username:<name> --password:<pass>`
  - On success sets `JWT_TOKEN` and `CURRENT_USER` in the process environment.
  - Prints count of pending requests.

- add — send a connection request
  - Usage: `add --username:<target>`

- view-requests — list pending connection requests
  - Usage: `view-requests`

- respond — accept or reject a connection request
  - Usage: `respond --username:<requester>`

- chat — start an encrypted chat session with an accepted connection
  - Usage: `chat --username:<target>`
  - Type messages; `exit` to quit.

Data Model (GORM)

- User: `id, username (unique), password (bcrypt), public_key, created_at`
- Connection: `id, sender_id, receiver_id, status('pending'|'accepted')`
- Message: `id, sender_id, receiver_id, content (encrypted), delivered, created_at`

How Messages Flow

1. Sender establishes a WebSocket with JWT.
2. Sender encrypts plaintext using receiver’s public key and sends Base64 ciphertext with `receiver_username`.
3. Server validates JWT, ensures a connection exists and is `accepted`, stores the encrypted message, relays to any online receiver sessions, and marks delivered.
4. If receiver is offline, message is stored; on reconnect, undelivered messages are pushed.

Local Keys

- Keep `keys/<username>_private.pem` secure. Losing it means you can’t decrypt past messages.
- Never upload your private key. Only the public key is sent to the server during registration.

Configuration Tips

- Change REST base URL: update `client/utils/utils.go`.
- Change WS URL: update `wsURL` in `client/commands/chat.go`.
- Ensure `JWT_SECRET` is set on the server; missing/empty secrets will break token validation.

Troubleshooting

- Login succeeds but commands fail: ensure server `JWT_SECRET` matches what the server uses for both issuing and validating.
- WebSocket refuses connection: confirm you include `Authorization: Bearer <token>`; client does this automatically.
- “User not found” in chat: ensure the target username exists and has accepted your connection request.
- Cannot decrypt messages: verify you are logged in as the same `CURRENT_USER` whose private key exists in `keys/` and that the PEM file format is correct.

License

MIT
