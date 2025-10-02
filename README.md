# 🎯 Stories Backend - Enterprise Ephemeral Stories Platform

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://docker.com)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-326CE5.svg)](https://kubernetes.io)
[![Production Ready](https://img.shields.io/badge/Production-Ready-success.svg)](#)

> **A production-ready, enterprise-grade backend for ephemeral stories** featuring real-time notifications, automatic expiration, social interactions, and comprehensive DevOps automation.


## ✨ **Features**

### 🎭 **Core Functionality**
- **Ephemeral Stories**: Auto-expiring content (1-168 hours)
- **Rich Media Support**: , images, and videos with MinIO/S3 storage
- **Story Privacy Controls**: Public, private, and friends-only visibility
- **Real-time Notifications**: WebSocket-based live updates (15+ event types)
- **Social Features**: Follow/unfollow, reactions (8 types), story views
- **User Profiles**: Complete user management with authentication

### 🔐 **Enterprise Security**
- **JWT Authentication**: Secure token-based auth with refresh tokens
- **Password Security**: Argon2 hashing with configurable complexity
- **Rate Limiting**: Redis-based sliding window with burst handling
- **Input Validation**: Comprehensive request validation and sanitization
- **CORS Protection**: Configurable cross-origin resource sharing
- **Session Management**: Secure session tracking and revocation

### ⚡ **Performance & Scalability**
- **Redis Caching**: Intelligent caching for high-performance reads
- **Database Optimization**: Strategic indexes and query optimization
- **Connection Pooling**: Efficient database connection management
- **Background Processing**: Async job queue for heavy operations
- **Horizontal Scaling**: Kubernetes auto-scaling (3-20 replicas)

### 🚀 **Production Ready**
- **Health Monitoring**: Comprehensive health checks with dependencies
- **Metrics Collection**: Prometheus metrics for observability
- **Structured Logging**: JSON logging with conual information
- **Graceful Shutdown**: Clean service termination
- **Multi-arch Docker**: AMD64 + ARM64 support
- **Complete CI/CD**: GitHub Actions automation

---

## 🚀 **Quick Start**

Get the Stories Backend running in **2 minutes**:

1. Clone the repo
```
cd stories-backend
```
2. Start all services (Docker required)
```
make setup-dev
```
```
curl http://localhost:8080/health
```
Your Stories Backend is now running!



**Access Points:**
- **API Server**: http://localhost:8080
- **API Documentation**: http://localhost:8080/api/v1
- **Health Check**: http://localhost:8080/health
- **Metrics**: http://localhost:9090/metrics
- **WebSocket**: ws://localhost:8080/ws

---

## 📋 **Requirements**

### **Development Environment**
- **Go**: 1.21+ ([Download](https://golang.org/dl/))
- **Docker**: 20.10+ ([Download](https://docs.docker.com/get-docker/))
- **Docker Compose**: 2.0+ ([Download](https://docs.docker.com/compose/install/))

### **Production Dependencies**
- **PostgreSQL**: 13+ (for data storage)
- **Redis**: 6+ (for caching & rate limiting)
- **MinIO/S3**: Compatible storage (for media files)

### **Optional Tools**
- **kubectl**: For Kubernetes deployment
- **k6**: For load testing
- **golangci-lint**: For code linting

---

## 🔧 **Installation**

### **Method 1: Automated Setup (Recommended)**

Clone and setup in one command
```
make setup-dev
```


### **Method 2: Manual Setup**

1. Copy environment configuration
```
cp .env.example .env
```
Edit .env with your settings
2. Start dependencies
```
make services-up
```
3. Run database migrations
```
make migrate-up
```
4. Seed sample data (optional)
```
make seed
```
5. Build and run
```
make build
./bin/stories-backend-api
```


### **Method 3: Docker Compose**

Start everything with Docker
```
docker-compose up -d
```
Check status
```
docker-compose ps
```
---

## ⚙️ **Configuration**

The application uses environment variables for configuration. Copy `.env.example` to `.env` and customize:

### **Essential Configuration**

Database
```
DATABASE_URL=postgresql://user:pass@localhost:5432/stories_db
```
Redis
```
REDIS_URL=redis://localhost:6379/0
```
JWT Security
```
JWT_SECRET=your-super-secure-256-bit-secret
JWT_REFRESH_SECRET=your-refresh-secret
```
MinIO/S3 Storage
```
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
```


### **Complete Configuration**
See [.env.example](.env.example) for all 130+ configuration options including:
- Server settings (port, timeouts, CORS)
- Database connection pooling
- Redis caching configuration
- Rate limiting settings
- Logging levels and formats
- Feature flags
- Security settings

---

## 📚 **API Documentation**

### **Authentication Endpoints**

POST /api/v1/auth/signup # Create account
POST /api/v1/auth/login # Login
POST /api/v1/auth/refresh # Refresh token
POST /api/v1/auth/logout # Logout



### **Story Management**

GET /api/v1/stories # Get stories feed
POST /api/v1/stories # Create story
GET /api/v1/stories/:id # Get specific story
PUT /api/v1/stories/:id # Update story
DELETE /api/v1/stories/:id # Delete story
POST /api/v1/stories/:id/view # Mark as viewed



### **Social Features**

POST /api/v1/stories/:id/reactions # Add reaction
GET /api/v1/users/:id/follow # Follow user
DELETE /api/v1/users/:id/follow # Unfollow user
GET /api/v1/users/search # Search users



### **Media Upload**

POST /api/v1/media/upload-url # Get presigned upload URL
GET /api/v1/media/:key # Get media file



### **Real-time WebSocket**

WebSocket: /ws?token=JWT_TOKEN # Real-time notifications



### **Example API Usage**

1. Create Account
```
curl -X POST http://localhost:8080/api/v1/auth/signup
-H "Content-Type: application/json"
-d '{
"email": "user@example.com",
"username": "testuser",
"password": "SecurePass123!",
"full_name": "Test User"
}'
```
2. Create Story
```
curl -X POST http://localhost:8080/api/v1/stories
-H "Authorization: Bearer YOUR_JWT_TOKEN"
-H "Content-Type: application/json"
-d '{
"type": "",
"": "My first story! 🎉",
"visibility": "public"
}'
```


---

## 🐳 **Deployment**

### **Development Deployment**

Start local development environment
```
make dev
```
Or with Docker Compose
```
docker-compose up -d
```


### **Production Deployment**

#### **Docker (Recommended)**

Build production images
```
make docker-build-prod
```
Deploy to production
```
make deploy-prod
```


#### **Kubernetes**

Deploy to Kubernetes cluster
```
make deploy-k8s
```
Scale horizontally
```
kubectl scale deployment stories-backend-api --replicas=10 -n stories-backend
```
Check status
```
make k8s-status
```


#### **Manual Deployment**

Build binaries
```
make build
```
Run API server
```
./bin/stories-backend-api
```
Run worker (separate terminal)
```
./bin/stories-backend-worker
```


---

## 📊 **Monitoring**

### **Health Checks**

Basic health
```
curl http://localhost:8080/health
```
Detailed health with dependencies
```
curl http://localhost:8080/health?detailed=true
```
Kubernetes probes
```
curl http://localhost:8080/health/ready # Readiness
curl http://localhost:8080/health/live # Liveness
```


### **Metrics & Observability**
- **Prometheus Metrics**: `http://localhost:9090/metrics`
- **Application Metrics**: HTTP requests, database stats, WebSocket connections
- **Business Metrics**: Story creation rates, user activity, reaction counts
- **Infrastructure Metrics**: CPU, memory, disk usage

### **Logging**
Structured JSON logs with conual information:

{
"level": "info",
"timestamp": "2025-01-15T10:30:45Z",
"msg": "Story created successfully",
"user_id": "123e4567-e89b-12d3-a456-426614174000",
"story_id": "789e4567-e89b-12d3-a456-426614174000",
"duration_ms": 45.2
}



---

## 🛠️ **Development**

### **Available Make Commands**

Development
```
make dev # Start with live reload
make build # Build binaries
make test # Run tests
make lint # Code linting
```
Database
```
make migrate-up # Apply migrations
make migrate-down # Rollback migrations
make seed # Add sample data
```
Docker
```
make docker-build # Build images
make docker-push # Push to registry
```
Deployment
```
make deploy-k8s # Deploy to Kubernetes
make health-check # Check service health
```


### **Development Workflow**

1. Setup development environment
```
make setup-dev
```
2. Start development server with live reload
```
make dev
```
3. Run tests
```
make test
```
4. Check code quality
```
make lint
```
5. Build for production
```
make build
```


### **Database Changes**

Create new migration
```
make migrate-create NAME=add_new_table
```
Apply migrations
```
make migrate-up
```
Rollback if needed
```
make migrate-down
```


---

## 🧪 **Testing**

### **Run Tests**

Unit tests
```
make test
```
Integration tests
```
make test-integration
```
Load tests
```
make test-load
```
All tests
```
make test-all
```


### **Test Categories**
- **Unit Tests**: Individual component testing
- **Integration Tests**: API endpoint testing with real dependencies
- **Load Tests**: Performance testing with k6
- **Security Tests**: OWASP security scanning

---

## 🔧 **Troubleshooting**

### **Common Issues**

#### **Service Won't Start**

Check dependencies
```
make services-up
make wait-for-services
```
Check configuration
```
make env-validate
```
View logs
```
make logs
```


#### **Database Issues**

Reset database
```
make db-reset
```
Check connection
```
make db-ping
```
Run migrations
```
make migrate-up
```


#### **Performance Issues**

Check health
```
make health-check
```
View metrics
```
curl http://localhost:9090/metrics
```
Check logs
```
make logs api
```