# Streak - Sports Streaming Platform System Design

## Executive Summary
Streak is a sports-oriented live streaming platform designed to connect athletes with their followers. This document outlines the architecture for an MVP that can scale to Twitch-level traffic while being deliverable in 4 weeks.

## Table of Contents
1. [Core Requirements](#core-requirements)
2. [High-Level Architecture](#high-level-architecture)
3. [Technology Stack](#technology-stack)
4. [System Components](#system-components)
5. [Database Design](#database-design)
6. [API Design](#api-design)
7. [Scaling Strategy](#scaling-strategy)
8. [MVP Feature Scope](#mvp-feature-scope)

## Core Requirements

### Functional Requirements (MVP)
- **User Management**: Athlete and follower accounts with profiles
- **Live Streaming**: Athletes can broadcast live streams
- **Stream Discovery**: Browse live and upcoming streams by sport/category
- **Real-time Chat**: Viewers can chat during live streams
- **Follow System**: Followers can subscribe to athlete channels
- **Stream Metadata**: Title, sport category, viewer count, duration

### Non-Functional Requirements
- **Low Latency**: <3 second glass-to-glass latency for live streams
- **Scalability**: Support 100K+ concurrent viewers initially, scale to millions
- **Availability**: 99.9% uptime SLA
- **Performance**: Sub-200ms API response times
- **Real-time**: Chat messages delivered in <100ms

### Future Enhancements (Post-MVP)
- VOD (Video on Demand) playback
- Clips and highlights
- Monetization (subscriptions, donations)
- Advanced analytics for athletes
- Mobile apps (iOS/Android)
- Multi-bitrate adaptive streaming

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CDN Layer (Cloudflare)                  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────┬──────────────────────┬──────────────────────┐
│   Web Frontend  │   Mobile App (Future)│   Streaming Apps     │
│   (React/Next)  │   (React Native)     │   (OBS, Streamlabs)  │
└─────────────────┴──────────────────────┴──────────────────────┘
         ↓                                          ↓
┌──────────────────────────────┐    ┌────────────────────────────┐
│   Load Balancer (nginx)      │    │   RTMP Ingest Servers      │
└──────────────────────────────┘    └────────────────────────────┘
         ↓                                          ↓
┌──────────────────────────────┐    ┌────────────────────────────┐
│   API Gateway                │    │   Streaming Pipeline       │
│   (Node.js/Express)          │    │   (Media Server)           │
└──────────────────────────────┘    └────────────────────────────┘
         ↓                                          ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Microservices Layer                          │
├──────────────┬──────────────┬──────────────┬───────────────────┤
│ Auth Service │Stream Service│ Chat Service │ Discovery Service │
└──────────────┴──────────────┴──────────────┴───────────────────┘
         ↓              ↓              ↓              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      Data Layer                                 │
├────────────────┬────────────────┬──────────────┬───────────────┤
│  PostgreSQL    │     Redis      │  MongoDB     │  S3 Storage   │
│  (Users, Meta) │  (Cache, Live) │(Chat History)│ (Thumbnails)  │
└────────────────┴────────────────┴──────────────┴───────────────┘
```

## Technology Stack

### Frontend
- **Framework**: Next.js 14 (React 18)
- **UI Library**: Tailwind CSS + shadcn/ui
- **Video Player**: Video.js with HLS.js
- **Real-time**: Socket.io-client
- **State Management**: Zustand
- **API Client**: Axios with SWR for caching

### Backend
- **Runtime**: Node.js 20 LTS
- **Framework**: Express.js
- **Language**: TypeScript
- **API Style**: RESTful + WebSocket for real-time

### Streaming Infrastructure
- **Ingest Protocol**: RTMP (via OBS/Streamlabs)
- **Media Server**: Node-Media-Server (RTMP) → FFmpeg → HLS
- **Delivery Protocol**: HLS (HTTP Live Streaming)
- **CDN**: Cloudflare Stream or self-hosted

### Data Storage
- **Primary Database**: PostgreSQL 15 (user data, stream metadata)
- **Cache Layer**: Redis 7 (sessions, live viewer counts, trending)
- **Chat Storage**: MongoDB (scalable document store for messages)
- **Object Storage**: S3-compatible (thumbnails, avatars, future VODs)

### Real-time Communication
- **Chat**: Socket.io (WebSocket with fallbacks)
- **Live Updates**: Server-Sent Events (SSE) for viewer counts

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Orchestration**: Kubernetes (for production scale)
- **CI/CD**: GitHub Actions
- **Monitoring**: Prometheus + Grafana
- **Logging**: Winston + ELK Stack (future)

## System Components

### 1. Authentication Service
**Responsibilities**:
- User registration and login (email/password, OAuth future)
- JWT token generation and validation
- Role management (athlete vs follower)
- Session management

**Technology**: Express + Passport.js + JWT
**Database**: PostgreSQL (users table)
**Cache**: Redis (session tokens, refresh tokens)

### 2. Stream Management Service
**Responsibilities**:
- Stream lifecycle (start, stop, status)
- Stream metadata (title, category, tags)
- Viewer count tracking
- Stream quality settings

**Technology**: Express + TypeScript
**Database**: PostgreSQL (streams, stream_sessions)
**Cache**: Redis (live stream status, viewer counts)

### 3. Video Streaming Pipeline
**Responsibilities**:
- RTMP ingest from streaming software
- Transcoding to HLS format
- Multi-bitrate encoding (future)
- Stream health monitoring

**Technology**:
- RTMP Server: Node-Media-Server
- Transcoding: FFmpeg
- HLS Serving: nginx or CDN

**Flow**:
```
OBS/Streamlabs (RTMP) → Node-Media-Server → FFmpeg Transcoding → HLS Segments → CDN → Viewers
```

### 4. Real-time Chat Service
**Responsibilities**:
- Message broadcasting to stream viewers
- Rate limiting and spam protection
- Chat moderation (bans, timeouts - future)
- Emote support (future)

**Technology**: Socket.io + Redis Pub/Sub
**Database**: MongoDB (chat history, optional for MVP)
**Scaling**: Redis Pub/Sub for multi-instance coordination

### 5. Discovery & Feed Service
**Responsibilities**:
- Live stream listing
- Search and filtering by sport category
- Trending streams algorithm
- Following feed

**Technology**: Express + PostgreSQL
**Cache**: Redis (trending calculations, feed cache)
**Search**: PostgreSQL full-text search (Elasticsearch future)

### 6. User Profile Service
**Responsibilities**:
- Athlete and follower profiles
- Follow/unfollow functionality
- Profile customization
- Statistics (followers, stream history)

**Technology**: Express + PostgreSQL
**Storage**: S3 (profile pictures, banners)

## Database Design

### PostgreSQL Schema

```sql
-- Users table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(20) NOT NULL CHECK (role IN ('athlete', 'follower')),
  display_name VARCHAR(100),
  bio TEXT,
  avatar_url VARCHAR(500),
  banner_url VARCHAR(500),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  is_verified BOOLEAN DEFAULT FALSE,
  is_banned BOOLEAN DEFAULT FALSE
);

-- Athlete profiles (extended info for athletes)
CREATE TABLE athlete_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  sport_categories VARCHAR(50)[] NOT NULL,
  stream_key VARCHAR(100) UNIQUE NOT NULL,
  rtmp_url VARCHAR(500),
  follower_count INTEGER DEFAULT 0,
  total_view_count BIGINT DEFAULT 0,
  is_live BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Streams table (metadata for each stream)
CREATE TABLE streams (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  athlete_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title VARCHAR(200) NOT NULL,
  description TEXT,
  sport_category VARCHAR(50) NOT NULL,
  tags VARCHAR(50)[],
  thumbnail_url VARCHAR(500),
  is_live BOOLEAN DEFAULT FALSE,
  started_at TIMESTAMP,
  ended_at TIMESTAMP,
  peak_viewer_count INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stream sessions (individual broadcast sessions)
CREATE TABLE stream_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
  started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ended_at TIMESTAMP,
  duration_seconds INTEGER,
  average_viewers INTEGER DEFAULT 0,
  peak_viewers INTEGER DEFAULT 0,
  total_messages INTEGER DEFAULT 0
);

-- Follows relationship
CREATE TABLE follows (
  follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  athlete_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (follower_id, athlete_id)
);

-- Indexes for performance
CREATE INDEX idx_streams_athlete_id ON streams(athlete_id);
CREATE INDEX idx_streams_is_live ON streams(is_live) WHERE is_live = TRUE;
CREATE INDEX idx_streams_sport_category ON streams(sport_category);
CREATE INDEX idx_follows_athlete_id ON follows(athlete_id);
CREATE INDEX idx_follows_follower_id ON follows(follower_id);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
```

### Redis Data Structures

```
# Live viewer counts (sorted set for leaderboards)
live:stream:{stream_id}:viewers -> Set of user IDs
live:stream:{stream_id}:viewer_count -> Integer

# Stream status cache
stream:{stream_id}:status -> JSON {is_live, viewer_count, started_at}

# Trending streams (sorted set by score)
trending:streams -> Sorted Set (stream_id, score)

# User sessions
session:{user_id} -> JWT token hash

# Rate limiting for chat
ratelimit:chat:{user_id}:{stream_id} -> Counter with TTL
```

### MongoDB Collections

```javascript
// Chat messages
{
  _id: ObjectId,
  stream_id: String,
  user_id: String,
  username: String,
  message: String,
  timestamp: Date,
  is_deleted: Boolean
}

// Indexes
db.messages.createIndex({ stream_id: 1, timestamp: -1 })
db.messages.createIndex({ user_id: 1 })
```

## API Design

### Authentication Endpoints

```
POST   /api/auth/register          - Create new user account
POST   /api/auth/login             - Login and get JWT token
POST   /api/auth/refresh           - Refresh access token
POST   /api/auth/logout            - Invalidate session
GET    /api/auth/me                - Get current user info
```

### User & Profile Endpoints

```
GET    /api/users/:userId          - Get user profile
PUT    /api/users/:userId          - Update user profile (auth required)
GET    /api/users/:userId/streams  - Get user's stream history
POST   /api/users/:userId/follow   - Follow a user
DELETE /api/users/:userId/follow   - Unfollow a user
GET    /api/users/:userId/followers- Get follower list
GET    /api/users/:userId/following- Get following list
```

### Stream Endpoints

```
GET    /api/streams                - Get live streams (paginated, filtered)
GET    /api/streams/:streamId      - Get stream details
POST   /api/streams                - Create new stream (athlete only)
PUT    /api/streams/:streamId      - Update stream metadata (athlete only)
DELETE /api/streams/:streamId      - End stream (athlete only)
GET    /api/streams/:streamId/stats- Get stream statistics
POST   /api/streams/start          - Start broadcasting (athlete only)
POST   /api/streams/stop           - Stop broadcasting (athlete only)
```

### Discovery Endpoints

```
GET    /api/discover/live          - Get live streams
GET    /api/discover/categories    - Get sport categories
GET    /api/discover/category/:cat - Get streams by category
GET    /api/discover/trending      - Get trending streams
GET    /api/discover/following     - Get live streams from followed athletes
GET    /api/search                 - Search streams and users
```

### Chat WebSocket Events

```
Client → Server:
- join_stream         - Join a stream's chat room
- leave_stream        - Leave a stream's chat room
- send_message        - Send a chat message

Server → Client:
- message             - New chat message
- viewer_count_update - Updated viewer count
- stream_ended        - Stream has ended
- user_banned         - User has been banned from chat
```

## Scaling Strategy

### Phase 1: MVP (Weeks 1-4)
**Target**: 1,000 concurrent viewers across 10-50 streams

**Infrastructure**:
- Single server deployment with Docker Compose
- PostgreSQL on same server
- Redis on same server
- MongoDB on same server or cloud (MongoDB Atlas)
- FFmpeg transcoding on same server
- HLS files served from local nginx

**Estimated Cost**: $50-100/month (single VPS)

### Phase 2: Early Growth (Months 2-3)
**Target**: 10,000 concurrent viewers across 100-500 streams

**Infrastructure**:
- Separate services into multiple servers
- Database server (PostgreSQL + Redis + MongoDB)
- API server cluster (2-3 nodes)
- Dedicated streaming servers (2-3 nodes)
- CDN for HLS delivery (Cloudflare or BunnyCDN)
- Load balancer (nginx)

**Optimizations**:
- Redis caching for hot data
- Database connection pooling
- HLS segment caching on CDN
- Horizontal scaling of API servers

**Estimated Cost**: $500-1,000/month

### Phase 3: Scale (Months 4-6)
**Target**: 100,000+ concurrent viewers across 1,000+ streams

**Infrastructure**:
- Kubernetes cluster for orchestration
- Dedicated streaming cluster (auto-scaling)
- Database replication (read replicas)
- Redis cluster (sharding)
- MongoDB sharding
- Multi-region CDN
- Separate chat service cluster

**Optimizations**:
- Geographic distribution (edge servers)
- Database partitioning by user/stream
- Adaptive bitrate streaming
- WebRTC for ultra-low latency (future)

**Estimated Cost**: $5,000-15,000/month

### Key Scaling Bottlenecks & Solutions

| Bottleneck | Solution |
|------------|----------|
| Database read load | Read replicas + Redis caching |
| Database write load | Sharding by user_id or stream_id |
| Streaming bandwidth | CDN integration (Cloudflare/BunnyCDN) |
| Transcoding CPU | Dedicated GPU servers or cloud transcoding |
| Chat message volume | Redis Pub/Sub + MongoDB sharding |
| API server load | Horizontal scaling + load balancing |
| WebSocket connections | Dedicated socket.io servers with Redis adapter |

## MVP Feature Scope

### Week 1: Foundation
- [ ] Project setup (monorepo structure)
- [ ] Database schema implementation
- [ ] Authentication system (register/login)
- [ ] Basic user profiles
- [ ] API gateway setup

### Week 2: Streaming Core
- [ ] RTMP ingest server setup
- [ ] FFmpeg transcoding pipeline
- [ ] HLS delivery mechanism
- [ ] Stream creation/management APIs
- [ ] Basic stream listing

### Week 3: Frontend & Chat
- [ ] Next.js frontend setup
- [ ] User authentication UI
- [ ] Video player integration (HLS playback)
- [ ] Real-time chat (Socket.io)
- [ ] Stream discovery page
- [ ] Athlete dashboard

### Week 4: Polish & Deploy
- [ ] Follow system
- [ ] Live viewer counts
- [ ] Stream thumbnails
- [ ] Search and filtering
- [ ] Deployment setup (Docker Compose)
- [ ] Documentation
- [ ] Testing and bug fixes

### MVP Features Summary

**Must Have**:
✅ User registration/login
✅ Athlete and follower roles
✅ RTMP stream ingest
✅ HLS video playback
✅ Real-time chat
✅ Stream discovery (browse live streams)
✅ Follow athletes
✅ Basic profiles
✅ Viewer counts

**Nice to Have** (if time permits):
⭐ Stream thumbnails auto-generation
⭐ Search functionality
⭐ Email notifications for live streams
⭐ Basic analytics dashboard

**Explicitly Out of Scope** (Post-MVP):
❌ VOD/replay functionality
❌ Clips and highlights
❌ Monetization
❌ Mobile apps
❌ Multi-bitrate streaming
❌ Advanced moderation tools
❌ Emotes and badges
❌ Stream scheduling

## Performance Targets

| Metric | Target (MVP) | Target (Scale) |
|--------|--------------|----------------|
| Stream latency | <5 seconds | <3 seconds |
| Chat message latency | <200ms | <100ms |
| API response time (p95) | <500ms | <200ms |
| Concurrent viewers | 1,000 | 100,000+ |
| Concurrent streams | 50 | 1,000+ |
| Database query time (p95) | <100ms | <50ms |
| Page load time | <2s | <1s |

## Security Considerations

### MVP Security Measures
- Password hashing with bcrypt (cost factor 10)
- JWT tokens with 15min expiry + refresh tokens
- HTTPS for all API communication
- CORS configuration
- Rate limiting on authentication endpoints
- Input validation and sanitization
- SQL injection prevention (parameterized queries)
- XSS protection (Content Security Policy)
- Stream key security (unique per athlete, regeneratable)

### Future Security Enhancements
- OAuth 2.0 (Google, Twitter, Discord)
- Two-factor authentication (2FA)
- DDoS protection (Cloudflare)
- WAF (Web Application Firewall)
- Encrypted HLS streams (AES-128)
- Content moderation AI
- Abuse reporting system

## Monitoring & Observability

### MVP Monitoring
- Application logging (Winston)
- Error tracking (console-based, Sentry future)
- Basic health check endpoints
- Database connection monitoring
- Stream health checks

### Future Monitoring
- Prometheus metrics
- Grafana dashboards
- Distributed tracing (Jaeger)
- Log aggregation (ELK stack)
- Real-time alerting (PagerDuty)
- Custom business metrics (DAU, stream hours, chat activity)

## Technology Alternatives Considered

| Component | Chosen | Alternatives | Rationale |
|-----------|--------|--------------|-----------|
| Video delivery | HLS | DASH, WebRTC, RTMP | Best browser support, CDN-friendly |
| Backend | Node.js/Express | Go, Python/FastAPI | Fast development, ecosystem, team familiarity |
| Database | PostgreSQL | MySQL, CockroachDB | ACID guarantees, JSON support, proven at scale |
| Chat storage | MongoDB | Cassandra, PostgreSQL | Schema flexibility, easy horizontal scaling |
| Cache | Redis | Memcached | Rich data structures, pub/sub support |
| Frontend | Next.js/React | Vue, Svelte, Angular | SSR support, largest ecosystem, best for scale |
| Real-time | Socket.io | Native WebSockets, Pusher | Fallback support, room management, established |
| Media server | Node-Media-Server | Nginx-RTMP, Wowza, Ant Media | Node.js integration, open-source, easy setup |

## Deployment Architecture

### MVP Deployment (Single Server)

```yaml
# docker-compose.yml structure
services:
  postgres:
    image: postgres:15
    volumes: [pgdata]

  redis:
    image: redis:7

  mongodb:
    image: mongo:7

  api:
    build: ./backend
    depends_on: [postgres, redis, mongodb]

  media-server:
    build: ./streaming
    ports: [1935:1935] # RTMP

  nginx:
    image: nginx:alpine
    volumes: [./nginx.conf, ./hls]
    ports: [80:80, 443:443]

  frontend:
    build: ./frontend
    depends_on: [api]
```

### Production Deployment (Kubernetes)
- Separate namespaces for services
- Horizontal Pod Autoscaling (HPA)
- Persistent volumes for databases
- Ingress controller (nginx-ingress)
- Cert-manager for TLS
- Helm charts for easy deployment

## Cost Estimates

### MVP (Month 1)
- VPS (8 cores, 16GB RAM): $50-80
- Domain + SSL: $15
- MongoDB Atlas (shared): $0 (free tier)
- Bandwidth (1TB): Included
- **Total: ~$70-100/month**

### Early Growth (Months 2-3)
- VPS cluster (3x): $150-240
- Database server: $100
- CDN (BunnyCDN): $50-200
- MongoDB Atlas: $50
- **Total: ~$350-600/month**

### Scale (Months 4+)
- Kubernetes cluster: $500-1000
- Database cluster: $300-500
- CDN bandwidth: $500-2000
- Media transcoding: $200-500
- Monitoring/logging: $100
- **Total: ~$1,600-4,000/month**

## Next Steps

1. **Week 1**: Set up development environment and implement core backend services
2. **Week 2**: Build streaming infrastructure and test RTMP → HLS pipeline
3. **Week 3**: Develop frontend and integrate real-time features
4. **Week 4**: Polish, test, deploy, and document

## Conclusion

This system design provides a pragmatic path to launching Streak as a sports-focused streaming platform. The architecture prioritizes:
- **Rapid MVP delivery** (4 weeks) while maintaining code quality
- **Clear scaling path** from hundreds to millions of users
- **Cost efficiency** at each stage of growth
- **Modern, proven technologies** with strong community support

The modular design allows for incremental improvements and feature additions post-launch without major refactoring.
