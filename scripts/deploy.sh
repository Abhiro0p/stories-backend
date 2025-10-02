#!/bin/bash

# Deployment Script for Stories Backend
# Handles deployment to various environments (Docker, Kubernetes, etc.)

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
ENVIRONMENT=${ENVIRONMENT:-"development"}
NAMESPACE=${NAMESPACE:-"stories-backend"}

# Docker configuration
DOCKER_REGISTRY=${DOCKER_REGISTRY:-"ghcr.io"}
DOCKER_NAMESPACE=${DOCKER_NAMESPACE:-"your-username"}

# Show deployment information
show_deploy_info() {
    echo -e "${BLUE}🚀 Stories Backend Deployment Script${NC}"
    echo -e "${YELLOW}Deployment Information:${NC}"
    echo -e "  Application: $APP_NAME"
    echo -e "  Version:     $VERSION"
    echo -e "  Environment: $ENVIRONMENT"
    echo -e "  Namespace:   $NAMESPACE"
    echo ""
}

# Show usage information
show_usage() {
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 docker [environment]     - Deploy with Docker Compose"
    echo "  $0 k8s [environment]        - Deploy to Kubernetes"
    echo "  $0 k8s-apply               - Apply all Kubernetes manifests"
    echo "  $0 k8s-delete              - Delete all Kubernetes resources"
    echo "  $0 health-check            - Check deployment health"
    echo "  $0 logs [service]          - Show service logs"
    echo "  $0 status                  - Show deployment status"
    echo ""
    echo -e "${YELLOW}Environments:${NC}"
    echo "  development - Local development environment"
    echo "  staging     - Staging environment"
    echo "  production  - Production environment"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0 docker production       # Deploy to production with Docker"
    echo "  $0 k8s staging            # Deploy to staging Kubernetes"
    echo "  $0 health-check           # Check if services are healthy"
}

# Deploy with Docker Compose
deploy_docker() {
    local env=${1:-"development"}
    
    echo -e "${BLUE}🐳 Deploying with Docker Compose (${env})...${NC}"
    
    # Select appropriate compose file
    local compose_file="docker-compose.yml"
    if [ "$env" = "production" ]; then
        compose_file="deployments/docker/docker-compose.prod.yml"
    fi
    
    # Check if compose file exists
    if [ ! -f "$compose_file" ]; then
        echo -e "${RED}❌ Compose file not found: $compose_file${NC}"
        exit 1
    fi
    
    # Pull latest images if in production
    if [ "$env" = "production" ]; then
        echo -e "${YELLOW}📥 Pulling latest images...${NC}"
        docker-compose -f "$compose_file" pull
    fi
    
    # Deploy services
    echo -e "${YELLOW}🚀 Starting services...${NC}"
    docker-compose -f "$compose_file" up -d
    
    # Wait for services to be ready
    echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
    sleep 30
    
    # Check service health
    check_docker_health "$compose_file"
    
    echo -e "${GREEN}✅ Docker deployment completed successfully!${NC}"
    show_docker_status "$compose_file"
}

# Check Docker service health
check_docker_health() {
    local compose_file=$1
    
    echo -e "${YELLOW}🏥 Checking service health...${NC}"
    
    # Check API service
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        echo -e "${GREEN}✅ API service is healthy${NC}"
    else
        echo -e "${RED}❌ API service is not responding${NC}"
    fi
    
    # Check database
    if docker-compose -f "$compose_file" exec -T postgres pg_isready -U stories_user >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Database is healthy${NC}"
    else
        echo -e "${RED}❌ Database is not responding${NC}"
    fi
    
    # Check Redis
    if docker-compose -f "$compose_file" exec -T redis redis-cli ping >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Redis is healthy${NC}"
    else
        echo -e "${RED}❌ Redis is not responding${NC}"
    fi
}

# Show Docker deployment status
show_docker_status() {
    local compose_file=$1
    
    echo -e "${BLUE}📊 Docker Deployment Status:${NC}"
    docker-compose -f "$compose_file" ps
}

# Deploy to Kubernetes
deploy_k8s() {
    local env=${1:-"development"}
    
    echo -e "${BLUE}☸️ Deploying to Kubernetes (${env})...${NC}"
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}❌ kubectl is not installed${NC}"
        exit 1
    fi
    
    # Check if cluster is accessible
    if ! kubectl cluster-info >/dev/null 2>&1; then
        echo -e "${RED}❌ Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi
    
    # Create namespace if it doesn't exist
    echo -e "${YELLOW}📦 Creating namespace...${NC}"
    kubectl apply -f deployments/k8s/namespace.yaml
    
    # Apply configurations
    echo -e "${YELLOW}⚙️ Applying configurations...${NC}"
    kubectl apply -f deployments/k8s/configmap.yaml
    kubectl apply -f deployments/k8s/secret.yaml
    
    # Apply deployments
    echo -e "${YELLOW}🚀 Applying deployments...${NC}"
    kubectl apply -f deployments/k8s/deployment.yaml
    
    # Apply services
    echo -e "${YELLOW}🔗 Applying services...${NC}"
    kubectl apply -f deployments/k8s/service.yaml
    
    # Apply ingress
    echo -e "${YELLOW}🌐 Applying ingress...${NC}"
    kubectl apply -f deployments/k8s/ingress.yaml
    
    # Apply HPA
    echo -e "${YELLOW}📈 Applying auto-scaling...${NC}"
    kubectl apply -f deployments/k8s/hpa.yaml
    
    # Wait for deployments to be ready
    echo -e "${YELLOW}⏳ Waiting for deployments to be ready...${NC}"
    kubectl wait --for=condition=available --timeout=300s deployment/stories-backend-api -n "$NAMESPACE"
    kubectl wait --for=condition=available --timeout=300s deployment/stories-backend-worker -n "$NAMESPACE"
    
    echo -e "${GREEN}✅ Kubernetes deployment completed successfully!${NC}"
    show_k8s_status
}

# Apply all Kubernetes manifests
k8s_apply_all() {
    echo -e "${BLUE}☸️ Applying all Kubernetes manifests...${NC}"
    
    local manifests=(
        "deployments/k8s/namespace.yaml"
        "deployments/k8s/configmap.yaml"
        "deployments/k8s/secret.yaml"
        "deployments/k8s/deployment.yaml"
        "deployments/k8s/service.yaml"
        "deployments/k8s/ingress.yaml"
        "deployments/k8s/hpa.yaml"
    )
    
    for manifest in "${manifests[@]}"; do
        if [ -f "$manifest" ]; then
            echo -e "${YELLOW}📋 Applying $manifest...${NC}"
            kubectl apply -f "$manifest"
        else
            echo -e "${YELLOW}⚠️ Manifest not found: $manifest${NC}"
        fi
    done
    
    echo -e "${GREEN}✅ All manifests applied successfully!${NC}"
}

# Delete all Kubernetes resources
k8s_delete_all() {
    echo -e "${RED}🗑️ Deleting all Kubernetes resources...${NC}"
    
    read -p "Are you sure you want to delete all resources in namespace '$NAMESPACE'? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
        echo -e "${GREEN}✅ All resources deleted${NC}"
    else
        echo -e "${YELLOW}❌ Deletion cancelled${NC}"
    fi
}

# Show Kubernetes deployment status
show_k8s_status() {
    echo -e "${BLUE}📊 Kubernetes Deployment Status:${NC}"
    
    echo -e "${YELLOW}Pods:${NC}"
    kubectl get pods -n "$NAMESPACE"
    
    echo -e "${YELLOW}Services:${NC}"
    kubectl get services -n "$NAMESPACE"
    
    echo -e "${YELLOW}Ingress:${NC}"
    kubectl get ingress -n "$NAMESPACE"
    
    echo -e "${YELLOW}HPA:${NC}"
    kubectl get hpa -n "$NAMESPACE"
}

# Check deployment health
check_health() {
    echo -e "${BLUE}🏥 Checking deployment health...${NC}"
    
    # Try different health check endpoints
    local endpoints=(
        "http://localhost:8080/health"
        "https://api.yourdomain.com/health"
    )
    
    for endpoint in "${endpoints[@]}"; do
        echo -e "${YELLOW}🔍 Checking $endpoint...${NC}"
        if curl -f "$endpoint" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ $endpoint is healthy${NC}"
            curl -s "$endpoint" | jq . 2>/dev/null || curl -s "$endpoint"
            return 0
        else
            echo -e "${RED}❌ $endpoint is not responding${NC}"
        fi
    done
    
    echo -e "${RED}❌ No healthy endpoints found${NC}"
    return 1
}

# Show service logs
show_logs() {
    local service=${1:-"api"}
    
    echo -e "${BLUE}📋 Showing logs for $service...${NC}"
    
    case $service in
        "api")
            if command -v kubectl &> /dev/null && kubectl get pods -n "$NAMESPACE" >/dev/null 2>&1; then
                kubectl logs -f deployment/stories-backend-api -n "$NAMESPACE"
            else
                docker-compose logs -f stories-api
            fi
            ;;
        "worker")
            if command -v kubectl &> /dev/null && kubectl get pods -n "$NAMESPACE" >/dev/null 2>&1; then
                kubectl logs -f deployment/stories-backend-worker -n "$NAMESPACE"
            else
                docker-compose logs -f stories-worker
            fi
            ;;
        *)
            echo -e "${RED}❌ Unknown service: $service${NC}"
            echo -e "${YELLOW}Available services: api, worker${NC}"
            exit 1
            ;;
    esac
}

# Show overall deployment status
show_status() {
    echo -e "${BLUE}📊 Overall Deployment Status:${NC}"
    
    # Check if Kubernetes is available
    if command -v kubectl &> /dev/null && kubectl cluster-info >/dev/null 2>&1; then
        echo -e "${GREEN}☸️ Kubernetes deployment detected${NC}"
        show_k8s_status
    elif command -v docker-compose &> /dev/null; then
        echo -e "${GREEN}🐳 Docker Compose deployment detected${NC}"
        show_docker_status "docker-compose.yml"
    else
        echo -e "${YELLOW}⚠️ No active deployment detected${NC}"
    fi
}

# Main script logic
main() {
    show_deploy_info
    
    local command=${1:-""}
    local environment=${2:-$ENVIRONMENT}
    
    case $command in
        "docker")
            deploy_docker "$environment"
            ;;
        "k8s")
            deploy_k8s "$environment"
            ;;
        "k8s-apply")
            k8s_apply_all
            ;;
        "k8s-delete")
            k8s_delete_all
            ;;
        "health-check")
            check_health
            ;;
        "logs")
            show_logs "$2"
            ;;
        "status")
            show_status
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
