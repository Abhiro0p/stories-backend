#!/bin/bash

# Build Script for Stories Backend
# Handles building binaries, Docker images, and deployment artifacts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="stories-backend"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GO_VERSION=$(go version | awk '{print $3}')

# Docker configuration
DOCKER_REGISTRY=${DOCKER_REGISTRY:-"ghcr.io"}
DOCKER_NAMESPACE=${DOCKER_NAMESPACE:-"your-username"}
DOCKER_TAG=${DOCKER_TAG:-$VERSION}

# Build flags
LDFLAGS="-w -s -X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME -X main.GoVersion=$GO_VERSION"
BUILD_FLAGS="-ldflags \"$LDFLAGS\" -trimpath"

# Show build information
show_build_info() {
    echo -e "${BLUE}🔨 Stories Backend Build Script${NC}"
    echo -e "${YELLOW}Build Information:${NC}"
    echo -e "  Version:     $VERSION"
    echo -e "  Commit:      $COMMIT"
    echo -e "  Build Time:  $BUILD_TIME"
    echo -e "  Go Version:  $GO_VERSION"
    echo ""
}

# Show usage information
show_usage() {
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 binary           - Build Go binaries only"
    echo "  $0 docker           - Build Docker images"
    echo "  $0 docker-push      - Build and push Docker images"
    echo "  $0 all              - Build binaries and Docker images"
    echo "  $0 clean            - Clean build artifacts"
    echo ""
    echo -e "${YELLOW}Environment Variables:${NC}"
    echo "  VERSION             - Build version (default: git describe)"
    echo "  DOCKER_REGISTRY     - Docker registry (default: ghcr.io)"
    echo "  DOCKER_NAMESPACE    - Docker namespace (default: your-username)"
    echo "  DOCKER_TAG          - Docker tag (default: VERSION)"
}

# Clean build artifacts
clean_build() {
    echo -e "${BLUE}🧹 Cleaning build artifacts...${NC}"
    
    # Clean Go build cache
    go clean -cache
    go clean -testcache
    go clean -modcache
    
    # Clean binary directory
    if [ -d "bin" ]; then
        rm -rf bin/
        echo -e "${GREEN}✅ Binary directory cleaned${NC}"
    fi
    
    # Clean Docker build cache
    if command -v docker &> /dev/null; then
        docker system prune -f
        echo -e "${GREEN}✅ Docker build cache cleaned${NC}"
    fi
    
    echo -e "${GREEN}✅ Clean completed${NC}"
}

# Build Go binaries
build_binaries() {
    echo -e "${BLUE}🔨 Building Go binaries...${NC}"
    
    # Create bin directory
    mkdir -p bin/
    
    # Build API server
    echo -e "${YELLOW}📦 Building API server...${NC}"
    eval "go build $BUILD_FLAGS -o bin/${APP_NAME}-api cmd/api/main.go"
    echo -e "${GREEN}✅ API server built: bin/${APP_NAME}-api${NC}"
    
    # Build worker
    echo -e "${YELLOW}📦 Building worker...${NC}"
    eval "go build $BUILD_FLAGS -o bin/${APP_NAME}-worker cmd/worker/main.go"
    echo -e "${GREEN}✅ Worker built: bin/${APP_NAME}-worker${NC}"
    
    # Show binary information
    echo -e "${BLUE}📊 Binary Information:${NC}"
    ls -lh bin/
    
    # Verify binaries
    echo -e "${YELLOW}🔍 Verifying binaries...${NC}"
    if ./bin/${APP_NAME}-api --version >/dev/null 2>&1; then
        echo -e "${GREEN}✅ API binary is valid${NC}"
    else
        echo -e "${YELLOW}⚠️ API binary version check failed${NC}"
    fi
    
    if ./bin/${APP_NAME}-worker --version >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Worker binary is valid${NC}"
    else
        echo -e "${YELLOW}⚠️ Worker binary version check failed${NC}"
    fi
}

# Build Docker images
build_docker() {
    echo -e "${BLUE}🐳 Building Docker images...${NC}"
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker is not installed${NC}"
        exit 1
    fi
    
    # Build API image
    echo -e "${YELLOW}📦 Building API Docker image...${NC}"
    docker build \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT="$COMMIT" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        -f deployments/docker/Dockerfile.api \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-api:${DOCKER_TAG}" \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-api:latest" \
        .
    echo -e "${GREEN}✅ API image built: ${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-api:${DOCKER_TAG}${NC}"
    
    # Build worker image
    echo -e "${YELLOW}📦 Building worker Docker image...${NC}"
    docker build \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT="$COMMIT" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        -f deployments/docker/Dockerfile.worker \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-worker:${DOCKER_TAG}" \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-worker:latest" \
        .
    echo -e "${GREEN}✅ Worker image built: ${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-worker:${DOCKER_TAG}${NC}"
    
    # Show image information
    echo -e "${BLUE}📊 Docker Images:${NC}"
    docker images | grep "${APP_NAME}"
}

# Push Docker images
push_docker() {
    echo -e "${BLUE}📤 Pushing Docker images...${NC}"
    
    # Push API image
    echo -e "${YELLOW}📤 Pushing API image...${NC}"
    docker push "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-api:${DOCKER_TAG}"
    docker push "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-api:latest"
    echo -e "${GREEN}✅ API image pushed${NC}"
    
    # Push worker image
    echo -e "${YELLOW}📤 Pushing worker image...${NC}"
    docker push "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-worker:${DOCKER_TAG}"
    docker push "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-worker:latest"
    echo -e "${GREEN}✅ Worker image pushed${NC}"
}

# Build multi-architecture images
build_multiarch() {
    echo -e "${BLUE}🏗️ Building multi-architecture Docker images...${NC}"
    
    # Create builder if it doesn't exist
    if ! docker buildx ls | grep -q stories-builder; then
        docker buildx create --name stories-builder --use
        docker buildx inspect --bootstrap
    fi
    
    # Build and push API image for multiple architectures
    echo -e "${YELLOW}📦 Building multi-arch API image...${NC}"
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT="$COMMIT" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        -f deployments/docker/Dockerfile.api \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-api:${DOCKER_TAG}" \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-api:latest" \
        --push \
        .
    
    # Build and push worker image for multiple architectures
    echo -e "${YELLOW}📦 Building multi-arch worker image...${NC}"
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT="$COMMIT" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        -f deployments/docker/Dockerfile.worker \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-worker:${DOCKER_TAG}" \
        -t "${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/${APP_NAME}-worker:latest" \
        --push \
        .
    
    echo -e "${GREEN}✅ Multi-architecture images built and pushed${NC}"
}

# Main script logic
main() {
    show_build_info
    
    local command=${1:-""}
    
    case $command in
        "binary")
            build_binaries
            ;;
        "docker")
            build_docker
            ;;
        "docker-push")
            build_docker
            push_docker
            ;;
        "multiarch")
            build_multiarch
            ;;
        "all")
            build_binaries
            build_docker
            ;;
        "clean")
            clean_build
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        "")
            show_usage
            ;;
        *)
            echo -e "${RED}❌ Unknown command: $command${NC}"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"
