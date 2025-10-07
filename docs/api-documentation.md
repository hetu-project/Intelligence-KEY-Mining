# Intelligence KEY Mining - API Documentation

This document provides comprehensive API documentation for all services in the Intelligence KEY Mining system.

## Table of Contents

1. [Overview](#overview)
2. [Miner Gateway Service](#miner-gateway-service)
3. [Points Service](#points-service)
4. [SBT Service](#sbt-service)
5. [Validator Service](#validator-service)
6. [Common Response Formats](#common-response-formats)
7. [Error Handling](#error-handling)

## Overview

The Intelligence KEY Mining system consists of four main microservices:

- **Miner Gateway Service** (Port: 8080) - Main API gateway for task submission and management
- **Points Service** (Port: 8087) - Handles points distribution and user point tracking
- **SBT Service** (Port: 8080) - Manages Soul Bound Token (SBT) registration and metadata
- **Validator Service** (Port: 8081) - Validates tasks and provides consensus

All services use RESTful APIs with JSON payloads and follow consistent response formats.

## API Overview

The following table provides a quick reference to all available APIs across all services:

| Service                   | Endpoint                                 | Method | Description                            | Link                                                        |
| ------------------------- | ---------------------------------------- | ------ | -------------------------------------- | ----------------------------------------------------------- |
| **Miner Gateway Service** |                                          |        |                                        |                                                             |
|                           | `/health`                                | GET    | Basic health check                     | [Health Endpoints](#health-endpoints)                       |
|                           | `/ready`                                 | GET    | Readiness check with DB connectivity   | [Health Endpoints](#health-endpoints)                       |
|                           | `/vlc/status`                            | GET    | Get Vector Logical Clock status        | [VLC Status Endpoints](#vlc-status-endpoints)               |
|                           | `/coordinator/status`                    | GET    | Get round coordinator status           | [Round Coordinator Endpoints](#round-coordinator-endpoints) |
|                           | `/coordinator/current-round`             | GET    | Get current round information          | [Round Coordinator Endpoints](#round-coordinator-endpoints) |
|                           | `/coordinator/history`                   | GET    | Get round history                      | [Round Coordinator Endpoints](#round-coordinator-endpoints) |
|                           | `/api/v1/tasks/submit`                   | POST   | Submit a new task for processing       | [Task Management](#task-management)                         |
|                           | `/api/v1/tasks/status/:id`               | GET    | Get task status by ID                  | [Task Management](#task-management)                         |
|                           | `/api/v1/tasks/user/:wallet`             | GET    | Get user's task history                | [Task Management](#task-management)                         |
|                           | `/api/v1/task-creation/create`           | POST   | Create a new Twitter task              | [Task Creation Management](#task-creation-management)       |
|                           | `/api/v1/task-creation/status/:id`       | GET    | Get task creation status               | [Task Creation Management](#task-creation-management)       |
|                           | `/api/v1/task-creation/user/:wallet`     | GET    | List user's task creations             | [Task Creation Management](#task-creation-management)       |
|                           | `/api/v1/task-creation/stats`            | GET    | Get task creation statistics           | [Task Creation Management](#task-creation-management)       |
| **Validator Service**     |                                          |        |                                        |                                                             |
|                           | `/health`                                | GET    | Basic health check                     | [Health Endpoints](#health-endpoints-3)                     |
|                           | `/ready`                                 | GET    | Readiness check                        | [Health Endpoints](#health-endpoints-3)                     |
|                           | `/api/v1/validate`                       | POST   | Validate a task submitted by a miner   | [Validation Management](#validation-management)             |
|                           | `/api/v1/config`                         | GET    | Get validator configuration            | [Validation Management](#validation-management)             |
| **SBT Service**           |                                          |        |                                        |                                                             |
|                           | `/health`                                | GET    | Basic health check                     | [Health Endpoints](#health-endpoints-2)                     |
|                           | `/api/v1/sbt/register`                   | POST   | Register a new user and mint SBT       | [SBT Management](#sbt-management)                           |
|                           | `/api/v1/sbt/dynamic/:wallet`            | GET    | Get dynamic metadata for NFT platforms | [SBT Management](#sbt-management)                           |
|                           | `/api/v1/sbt/profile/:wallet`            | GET    | Get user profile information           | [SBT Management](#sbt-management)                           |
|                           | `/api/v1/sbt/profile/:wallet`            | PUT    | Update user profile                    | [SBT Management](#sbt-management)                           |
|                           | `/api/v1/sbt/invite/:wallet`             | PUT    | Update invitation relationships        | [SBT Management](#sbt-management)                           |
| **Points Service**        |                                          |        |                                        |                                                             |
|                           | `/health`                                | GET    | Basic health check                     | [Health Endpoints](#health-endpoints-1)                     |
|                           | `/api/v1/points/distribute`              | POST   | Distribute points to users             | [Points Management](#points-management)                     |
|                           | `/api/v1/points/user/:wallet_address`    | GET    | Get user's total points                | [Points Management](#points-management)                     |
|                           | `/api/v1/points/history/:wallet_address` | GET    | Get user's points history              | [Points Management](#points-management)                     |
|                           | `/api/v1/points/stats`                   | GET    | Get points system statistics           | [Points Management](#points-management)                     |
|                           | `/api/v1/points/config`                  | GET    | Get points configuration               | [Points Management](#points-management)                     |
|                           | `/api/v1/points/config`                  | PUT    | Update points configuration            | [Points Management](#points-management)                     |
|                           | `/api/v1/points/test`                    | POST   | Test points distribution (dev only)    | [Points Management](#points-management)                     |

## Miner Gateway Service

**Base URL:** `http://localhost:8080`

### Health Endpoints

#### GET /health

Basic health check endpoint.

**Response:**

```json
{
  "status": "ok",
  "service": "miner-gateway"
}
```

#### GET /ready

Readiness check endpoint (includes database connectivity check).

**Response:**

```json
{
  "status": "ready",
  "service": "miner-gateway"
}
```

### VLC Status Endpoints

#### GET /vlc/status

Get Vector Logical Clock (VLC) status.

**Response:**

```json
{
  "process_id": "miner-1",
  "clock": {
    "process_id": "miner-1",
    "logical_time": 123
  }
}
```

### Round Coordinator Endpoints

#### GET /coordinator/status

Get round coordinator status.

**Response:**

```json
{
  "current_round": {
    "round_id": "round_123",
    "status": "active",
    "start_time": "2024-01-01T00:00:00Z"
  },
  "is_active": true
}
```

#### GET /coordinator/current-round

Get current round information.

**Response:**

```json
{
  "current_round": {
    "round_id": "round_123",
    "status": "active",
    "start_time": "2024-01-01T00:00:00Z",
    "end_time": "2024-01-01T01:00:00Z"
  }
}
```

#### GET /coordinator/history

Get round history.

**Query Parameters:**

- `limit` (optional): Number of rounds to return (1-100, default: 10)

**Response:**

```json
{
  "rounds": [
    {
      "round_id": "round_123",
      "status": "completed",
      "start_time": "2024-01-01T00:00:00Z",
      "end_time": "2024-01-01T01:00:00Z"
    }
  ]
}
```

### API v1 Endpoints

#### Task Management

##### POST /api/v1/tasks/submit

Submit a new task for processing.

**Request Body:**

```json
{
  "user_wallet": "0x1234...abcd",
  "task_type": "twitter_retweet",
  "payload": {
    "tweet_id": "1234567890",
    "twitter_username": "example_user"
  }
}
```

**Response:**

```json
{
  "success": true,
  "task_id": "task_123",
  "message": "Task submitted successfully",
  "vlc_value": 5
}
```

##### GET /api/v1/tasks/status/:id

Get task status by ID.

**Path Parameters:**

- `id`: Task ID

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "task_123",
    "user_wallet": "0x1234...abcd",
    "task_type": "twitter_retweet",
    "status": "completed",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:05:00Z",
    "completed_at": "2024-01-01T00:05:00Z",
    "payload": {...},
    "vlc_clock": {...},
    "proof": {...}
  }
}
```

##### GET /api/v1/tasks/user/:wallet

Get user's task history.

**Path Parameters:**

- `wallet`: User wallet address

**Query Parameters:**

- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (1-100, default: 20)

**Response:**

```json
{
  "success": true,
  "data": {
    "tasks": [...],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 100
    }
  }
}
```

#### Task Creation Management

##### POST /api/v1/task-creation/create

Create a new Twitter task.

**Request Body:**

```json
{
  "user_wallet": "0x1234...abcd",
  "task_type": "twitter_retweet",
  "project_name": "Example Project",
  "project_icon": "https://example.com/icon.png",
  "description": "Project description",
  "twitter_username": "example_user",
  "twitter_link": "https://twitter.com/example_user/status/1234567890",
  "tweet_id": "1234567890"
}
```

**Response:**

```json
{
  "success": true,
  "task_id": "task_123",
  "message": "Task created successfully",
  "vlc_value": 3
}
```

##### GET /api/v1/task-creation/status/:id

Get task creation status.

**Path Parameters:**

- `id`: Task ID

**Response:**

```json
{
  "success": true,
  "task_id": "task_123",
  "status": "completed",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:05:00Z",
  "completed_at": "2024-01-01T00:05:00Z",
  "payload": {...},
  "vlc_value": 3,
  "vlc_clock": {...},
  "proof": {...}
}
```

##### GET /api/v1/task-creation/user/:wallet

List user's task creations.

**Path Parameters:**

- `wallet`: User wallet address

**Query Parameters:**

- `limit` (optional): Number of records (1-100, default: 50)
- `offset` (optional): Offset for pagination (0-10000, default: 0)

**Response:**

```json
{
  "success": true,
  "data": {
    "tasks": [...],
    "total": 25,
    "limit": 50,
    "offset": 0
  }
}
```

##### GET /api/v1/task-creation/stats

Get task creation statistics.

**Query Parameters:**

- `user_wallet` (optional): Filter by specific user

**Response:**

```json
{
  "success": true,
  "data": {
    "total_tasks": 1000,
    "completed_tasks": 950,
    "pending_tasks": 50,
    "user_stats": {...}
  }
}
```

## Points Service

**Base URL:** `http://localhost:8087`

### Health Endpoints

#### GET /health

Basic health check endpoint.

**Response:**

```json
{
  "status": "healthy",
  "service": "points-service"
}
```

### API v1 Endpoints

#### Points Management

##### POST /api/v1/points/distribute

Distribute points to users based on task completion.

**Request Body:**

```json
{
  "batch_id": "batch_123",
  "trigger_type": "validator_voting",
  "tasks": [
    {
      "user_wallet": "0x1234...abcd",
      "task_type": "creation",
      "vlc_value": 2,
      "task_id": "task_1"
    },
    {
      "user_wallet": "0x5678...efgh",
      "task_type": "retweet",
      "vlc_value": 3,
      "task_id": "task_2"
    }
  ]
}
```

**Response:**

```json
{
  "status": "success",
  "data": {
    "batch_id": "batch_123",
    "total_points_distributed": 100,
    "user_allocations": [
      {
        "user_wallet": "0x1234...abcd",
        "points": 40
      },
      {
        "user_wallet": "0x5678...efgh",
        "points": 60
      }
    ]
  }
}
```

##### GET /api/v1/points/user/:wallet_address

Get user's total points.

**Path Parameters:**

- `wallet_address`: User wallet address

**Response:**

```json
{
  "status": "success",
  "data": {
    "wallet_address": "0x1234...abcd",
    "total_points": 150
  }
}
```

##### GET /api/v1/points/history/:wallet_address

Get user's points history.

**Path Parameters:**

- `wallet_address`: User wallet address

**Query Parameters:**

- `limit` (optional): Number of records (default: 50)

**Response:**

```json
{
  "status": "success",
  "data": {
    "wallet_address": "0x1234...abcd",
    "history": [
      {
        "points": 50,
        "reason": "task_completion",
        "timestamp": "2024-01-01T00:00:00Z",
        "task_id": "task_123"
      }
    ],
    "count": 1
  }
}
```

##### GET /api/v1/points/stats

Get points system statistics.

**Response:**

```json
{
  "status": "success",
  "data": {
    "total_users": 1000,
    "total_points_distributed": 50000,
    "average_points_per_user": 50,
    "points_by_type": {
      "creation": 30000,
      "retweet": 20000
    }
  }
}
```

##### GET /api/v1/points/config

Get points configuration.

**Response:**

```json
{
  "status": "success",
  "data": {
    "total_pool_points": 100000,
    "creation_ratio": 0.6,
    "retweet_ratio": 0.4,
    "history_limit": 100
  }
}
```

##### PUT /api/v1/points/config

Update points configuration.

**Request Body:**

```json
{
  "total_pool_points": 100000,
  "creation_ratio": 0.6,
  "retweet_ratio": 0.4,
  "history_limit": 100
}
```

**Response:**

```json
{
  "status": "success",
  "message": "Configuration updated successfully",
  "data": {...}
}
```

##### POST /api/v1/points/test

Test points distribution (development only).

**Response:**

```json
{
  "status": "success",
  "message": "Test distribution completed",
  "data": {...}
}
```

## SBT Service

**Base URL:** `http://localhost:8080`

### Health Endpoints

#### GET /health

Basic health check endpoint.

**Response:**

```json
{
  "status": "ok",
  "service": "sbt-service",
  "timestamp": 1640995200
}
```

### API v1 Endpoints

#### SBT Management

##### POST /api/v1/sbt/register

Register a new user and mint SBT.

**Request Body:**

```json
{
  "wallet_address": "0x1234...abcd",
  "display_name": "John Doe",
  "avatar_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "image_url": "https://example.com/avatar.png",
  "invite_from": "0x5678...efgh",
  "invite_to": ["0x9999...1111", "0x8888...2222"]
}
```

**Response:**

```json
{
  "success": true,
  "wallet_address": "0x1234...abcd",
  "token_id": "123",
  "token_uri": "https://api.example.com/metadata/123",
  "contract_address": "0xContractAddress",
  "metadata": {
    "name": "John Doe's SBT",
    "description": "Soul Bound Token for John Doe",
    "image": "https://api.example.com/image/123",
    "attributes": [...]
  }
}
```

##### GET /api/v1/sbt/dynamic/:wallet

Get dynamic metadata for NFT platform calls.

**Path Parameters:**

- `wallet`: User wallet address

**Response:**

```json
{
  "name": "John Doe's SBT",
  "description": "Soul Bound Token for John Doe",
  "image": "https://api.example.com/image/123",
  "attributes": [
    {
      "trait_type": "Level",
      "value": "5"
    },
    {
      "trait_type": "Points",
      "value": "150"
    }
  ]
}
```

##### GET /api/v1/sbt/profile/:wallet

Get user profile information.

**Path Parameters:**

- `wallet`: User wallet address

**Response:**

```json
{
  "wallet_address": "0x1234...abcd",
  "display_name": "John Doe",
  "avatar_url": "https://api.example.com/image/123",
  "points": 150,
  "level": 5,
  "invite_from": "0x5678...efgh",
  "invite_to": ["0x9999...1111", "0x8888...2222"],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

##### PUT /api/v1/sbt/profile/:wallet

Update user profile.

**Path Parameters:**

- `wallet`: User wallet address

**Request Body:**

```json
{
  "display_name": "John Smith",
  "avatar_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "image_url": "https://example.com/new-avatar.png"
}
```

**Response:**

```json
{
  "message": "Profile updated successfully"
}
```

##### PUT /api/v1/sbt/invite/:wallet

Update invitation relationships.

**Path Parameters:**

- `wallet`: User wallet address

**Request Body:**

```json
{
  "invite_from": "0x5678...efgh",
  "invite_to": ["0x9999...1111", "0x8888...2222"]
}
```

**Response:**

```json
{
  "status": "success",
  "message": "Invite relation updated successfully"
}
```

## Validator Service

**Base URL:** `http://localhost:8081`

### Health Endpoints

#### GET /health

Basic health check endpoint.

**Response:**

```json
{
  "status": "ok",
  "service": "validator"
}
```

#### GET /ready

Readiness check endpoint.

**Response:**

```json
{
  "status": "ready",
  "service": "validator"
}
```

### API v1 Endpoints

#### Validation Management

##### POST /api/v1/validate

Validate a task submitted by a miner.

**Request Body:**

```json
{
  "event_id": "event_123",
  "task_id": "task_123",
  "miner_id": "miner-1",
  "task_type": "twitter_retweet",
  "payload": {
    "tweet_id": "1234567890",
    "twitter_username": "example_user"
  },
  "vlc_clock": {
    "process_id": "miner-1",
    "logical_time": 123
  }
}
```

**Response:**

```json
{
  "success": true,
  "vote": "approve",
  "score": 0.85,
  "weight": 1.0,
  "reasoning": "Task validation passed with high confidence"
}
```

##### GET /api/v1/config

Get validator configuration.

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "validator-1",
    "role": "primary",
    "weight": 1.0
  }
}
```

## Common Response Formats

### Success Response

```json
{
  "success": true,
  "data": {...},
  "message": "Operation completed successfully"
}
```

### Error Response

```json
{
  "success": false,
  "error": "Error description",
  "details": "Detailed error information"
}
```

### Pagination Response

```json
{
  "success": true,
  "data": {
    "items": [...],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

## Error Handling

All services follow consistent error handling patterns:

### HTTP Status Codes

- `200 OK` - Successful operation
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request format or parameters
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource already exists
- `500 Internal Server Error` - Server error

### Error Response Format

```json
{
  "success": false,
  "error": "Error type",
  "message": "Human-readable error message",
  "details": "Technical error details (optional)"
}
```

### Common Error Types

- `Validation failed` - Request validation failed
- `Resource not found` - Requested resource doesn't exist
- `Resource already exists` - Resource with same identifier already exists
- `Invalid request format` - Request body format is invalid
- `Database error` - Database operation failed
- `External service error` - External service call failed

## Authentication

Currently, the services do not implement authentication. All endpoints are publicly accessible. In production environments, appropriate authentication and authorization mechanisms should be implemented.

## Rate Limiting

The Miner Gateway Service implements rate limiting middleware to prevent abuse. Rate limits are applied per IP address and endpoint.

## CORS

All services implement CORS (Cross-Origin Resource Sharing) to allow requests from web applications. The CORS configuration allows:

- All origins (`*`)
- Methods: GET, POST, PUT, DELETE, OPTIONS
- Headers: Origin, Content-Type, Accept, Authorization, X-Requested-With

## Environment Variables

Each service requires specific environment variables for configuration:

### Miner Gateway Service

- `PORT` - Server port (default: 8080)
- `DATABASE_URL` - MySQL database connection string
- `MINER_PRIVATE_KEY` - Miner's private key
- `VALIDATOR_ENDPOINTS` - JSON array of validator endpoints
- `POINTS_SERVICE_URL` - Points service URL
- `TWITTER_RETWEET_CHECK_URL` - Twitter verification service URL

### Points Service

- `PORT` - Server port (default: 8080)
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 3306)
- `DB_USER` - Database username (default: root)
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name (default: hetu_key_mining)

### SBT Service

- `PORT` - Server port (default: 8080)
- `DATABASE_URL` - MySQL database connection string
- `PINATA_API_KEY` - Pinata API key for IPFS
- `PINATA_SECRET_KEY` - Pinata secret key
- `BASE_URL` - Base URL for the service
- `POINTS_SERVICE_URL` - Points service URL
- `REFERRAL_API_URL` - Referral API URL

### Validator Service

- `PORT` - Server port (default: 8081)
- `DATABASE_URL` - MySQL database connection string
- `VALIDATOR_ID` - Validator identifier
- `VALIDATOR_ROLE` - Validator role (primary/secondary)
- `VALIDATOR_WEIGHT` - Validator weight
- `VALIDATOR_PRIVATE_KEY` - Validator's private key

## Notes

1. All timestamps are in ISO 8601 format (UTC)
2. Wallet addresses must be valid Ethereum addresses (42 characters, starting with 0x)
3. Task types are case-sensitive and must match exactly
4. VLC (Vector Logical Clock) values are used for ordering and consistency
5. The system implements a round-based consensus mechanism for task validation
6. Points are distributed based on VLC values and task completion
7. SBT tokens are non-transferable and represent user identity and achievements
