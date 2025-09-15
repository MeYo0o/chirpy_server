# Chirpy Server 🐦

A production-ready JSON API server built with Go that mimics Twitter's core functionality. This project demonstrates modern backend development practices including authentication, authorization, database management, and webhook integration.

## What is Chirpy?

Chirpy is a microblogging platform API that allows users to:

- **Create and manage user accounts** with secure authentication
- **Post short messages (chirps)** up to 140 characters
- **Follow other users** and view their posts
- **Upgrade to premium accounts** via payment webhooks
- **Manage authentication tokens** with JWT and refresh tokens

Think of it as a simplified Twitter API built from scratch using Go's standard library and modern web development practices.

## Key Features

### 🔐 Authentication & Authorization

- **JWT-based authentication** with access and refresh tokens
- **Password hashing** using bcrypt for security
- **Token refresh mechanism** for seamless user experience
- **Role-based access control** (regular users vs premium users)

### 📝 Content Management

- **Create, read, update, and delete chirps** (posts)
- **Content filtering** to prevent inappropriate language
- **Character limit enforcement** (140 characters max)
- **User-specific chirp filtering** and sorting

### 💳 Payment Integration

- **Polka payment gateway integration** via webhooks
- **Premium account upgrades** (Chirpy Red)
- **Webhook event handling** for payment processing

### 📊 Monitoring & Administration

- **Request metrics tracking** with atomic counters
- **Admin dashboard** for monitoring server health
- **Database reset functionality** for development

## Why Use This Project?

This project serves as an excellent learning resource for:

- **Backend developers** wanting to understand HTTP servers in Go
- **Students** learning modern web API development
- **Developers** interested in authentication patterns and JWT implementation
- **Anyone** looking to understand database integration with Go
- **Teams** needing a reference implementation for microblogging APIs

## Technology Stack

- **Language**: Go 1.25+
- **Database**: PostgreSQL with SQLC for type-safe queries
- **Authentication**: JWT tokens with refresh mechanism
- **Password Hashing**: bcrypt
- **Database Migrations**: Goose
- **Environment Management**: godotenv
- **HTTP Server**: Go's standard library `net/http`

## Installation & Setup

### Prerequisites

- Go 1.25 or higher
- PostgreSQL database
- Git

### 1. Clone the Repository

```bash
git clone https://github.com/MeYo0o/chirpy_server.git
cd chirpy_server
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Database Setup

Create a PostgreSQL database and run migrations:

```bash
# Install goose (if not already installed)
go install github.com/pressly/goose/v3/cmd/goose@latest

# Set up your database connection
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="postgres://username:password@localhost/dbname?sslmode=disable"

# Run migrations
goose -dir sql/schema up
```

### 4. Environment Configuration

Copy the example environment file and configure it:

```bash
cp example.env .env
```

Edit `.env` with your configuration:

```env
# General
PLATFORM="dev"
DB_URL="postgres://username:password@localhost/chirpy_db?sslmode=disable"

# Generate a secure JWT secret (64 characters)
JWT_SECRET="your-64-character-secret-key-here"

# Payment Gateway (optional for webhook testing)
POLKA_KEY="your-polka-api-key"
```

Generate a secure JWT secret:

```bash
openssl rand -base64 64
```

### 5. Run the Server

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Authentication

- `POST /api/users` - Create a new user account
- `PUT /api/users` - Update user information
- `POST /api/login` - Login and get access token
- `POST /api/refresh` - Refresh access token
- `POST /api/revoke` - Revoke refresh token

### Chirps (Posts)

- `GET /api/chirps` - Get all chirps (with optional filtering)
- `POST /api/chirps` - Create a new chirp
- `GET /api/chirps/{id}` - Get a specific chirp
- `DELETE /api/chirps/{id}` - Delete a chirp (author only)

### Webhooks

- `POST /api/polka/webhooks` - Handle payment webhooks

### Admin

- `GET /admin/metrics` - View server metrics
- `POST /admin/reset` - Reset database (dev only)

### Health Check

- `GET /api/healthz` - Server health status

## Testing the API

A complete Postman collection is included in `docs/Chirpy.postman_collection.json` with all API endpoints and example requests.

### Quick Test with curl

1. **Create a user:**

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

2. **Login:**

```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

3. **Create a chirp:**

```bash
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{"body":"Hello, Chirpy!"}'
```

## Project Structure

```
chirpy_server/
├── main.go                 # Application entry point and routing
├── handlers.go             # HTTP request handlers
├── middlewares.go          # Custom middleware functions
├── internal/
│   ├── auth/              # Authentication utilities
│   └── database/          # Database models and queries
├── sql/
│   ├── schema/            # Database migrations
│   └── queries/           # SQLC query definitions
├── docs/
│   └── Chirpy.postman_collection.json
└── assets/
    └── logo.png
```

## Learning Outcomes

This project demonstrates key concepts from the "Learn HTTP Servers in Go" course:

### 1. **Servers** 🖥️

Understanding web server fundamentals and why Go excels at building performant HTTP servers with its excellent concurrency model and standard library.

### 2. **Routing** 🛣️

Implementation of HTTP routing using Go's standard library, including path parameters, query strings, and HTTP method handling.

### 3. **Architecture** 🏗️

Clean separation of concerns with handlers, middleware, database layer, and authentication modules following Go best practices.

### 4. **JSON** 📄

Comprehensive JSON handling including request parsing, response formatting, and error handling with proper HTTP status codes.

### 5. **Storage** 🗄️

PostgreSQL integration with SQLC for type-safe database queries, migrations with Goose, and proper database connection management.

### 6. **Authentication** 🔐

Complete JWT-based authentication system with access tokens, refresh tokens, password hashing using bcrypt, and secure token validation.

### 7. **Authorization** 🛡️

Role-based access control, resource ownership validation, and proper authorization checks for protected endpoints.

### 8. **Webhooks** 🔗

Payment gateway integration via webhooks, event handling, and external service communication patterns.

### 9. **Documentation** 📚

Comprehensive API documentation, Postman collection, and code documentation following Go conventions.

## Development Features

- **Type-safe database queries** with SQLC
- **Atomic request counting** for metrics
- **Graceful error handling** with proper HTTP status codes
- **Content filtering** for inappropriate language
- **Environment-based configuration**
- **Database migrations** for schema management
- **Comprehensive logging** and monitoring

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is for educational purposes and demonstrates modern Go web development practices.

---

**Built with ❤️ using Go** - A production-ready microblogging API that showcases the power and simplicity of Go for backend development.
