# Mobile Backend API - Firebase Integration Example

A Backend-as-a-Service example demonstrating Specular's spec-first development for building mobile application backends with Firebase integration.

## Project Type

This is an **api-service** template example showcasing:
- Firebase Authentication with social login
- Push notifications via Firebase Cloud Messaging
- Real-time data sync with Firestore
- File storage with Cloud Storage
- WebSocket connections for live updates

## Getting Started

```bash
cd examples/projects/mobile-backend
specular init --template api-service

# Generate and execute
specular plan create
specular build run --dry-run
```

## Prerequisites

- Go 1.22+
- Firebase project with Admin SDK credentials
- Service account JSON key file

## Environment Variables

```bash
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_CREDENTIALS_FILE=/path/to/service-account.json
FIREBASE_STORAGE_BUCKET=your-bucket.appspot.com
```

## Spec Overview

The specification defines:
- User authentication (email/password + social)
- Push notifications (iOS/Android)
- Real-time sync with conflict resolution
- File uploads with signed URLs
- User preferences and remote config

## Files

- `spec.yaml` - Product specification
- `.specular/` - Generated configuration

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login with email/password
- `POST /api/auth/social` - Social login (Google/Apple/Facebook)
- `POST /api/auth/refresh` - Refresh access token
- `GET /api/auth/profile` - Get current user profile
- `PUT /api/auth/profile` - Update user profile

### Push Notifications
- `POST /api/notifications/register` - Register device token
- `POST /api/notifications/send` - Send notification to user
- `POST /api/notifications/topic` - Broadcast to topic
- `DELETE /api/notifications/unregister` - Remove device token

### Real-time Sync
- `GET /api/sync/{collection}` - Get documents since timestamp
- `POST /api/sync/{collection}` - Sync documents with conflict detection
- `WS /api/sync/ws` - WebSocket for live updates

### File Storage
- `POST /api/files/upload` - Upload file
- `GET /api/files/{fileId}` - Get file metadata
- `DELETE /api/files/{fileId}` - Delete file
- `POST /api/files/signed-url` - Generate signed upload/download URLs

### Configuration
- `GET /api/users/{userId}/preferences` - Get user preferences
- `PUT /api/users/{userId}/preferences` - Update preferences
- `GET /api/config` - Get app configuration and feature flags

## Architecture

```
mobile-backend/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── handlers/             # HTTP handlers
│   │   ├── auth.go
│   │   ├── notifications.go
│   │   ├── sync.go
│   │   ├── files.go
│   │   └── preferences.go
│   ├── firebase/             # Firebase SDK wrappers
│   │   ├── auth.go
│   │   ├── messaging.go
│   │   ├── firestore.go
│   │   └── storage.go
│   ├── middleware/           # HTTP middleware
│   │   ├── auth.go
│   │   └── ratelimit.go
│   └── websocket/            # WebSocket handlers
│       └── sync.go
├── spec.yaml
└── .specular/
```

## Key Features

### Offline-First Sync
- Incremental synchronization using timestamps
- Conflict detection with last-write-wins strategy
- Batch document operations

### Secure File Uploads
- Pre-signed URLs for direct client uploads
- Server-side file validation
- Automatic thumbnail generation

### Multi-Platform Push
- FCM for Android, APNs via FCM for iOS
- Topic-based broadcasting
- Delivery tracking

## Firebase Setup

1. Create a Firebase project at [console.firebase.google.com](https://console.firebase.google.com)
2. Enable Authentication, Firestore, Cloud Messaging, and Storage
3. Generate a service account key (Project Settings > Service Accounts)
4. Download the JSON key file and set `FIREBASE_CREDENTIALS_FILE`

## Security Considerations

- All endpoints (except auth) require Firebase ID token
- Rate limiting prevents abuse
- File uploads validated by type and size
- WebSocket connections authenticated on connect
