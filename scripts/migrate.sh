#!/bin/bash

# Database Migration Script for Stories Backend
# Handles database migrations using golang-migrate

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DATABASE_URL=${DATABASE_URL:-"postgresql://stories_user:stories_pass@localhost:5432/stories_db?sslmode=disable"}
MIGRATIONS_DIR=${MIGRATIONS_DIR:-"./migrations"}
MIGRATE_CMD=${MIGRATE_CMD:-"migrate"}

# Check if migrate command exists
check_migrate_command() {
    if ! command -v "$MIGRATE_CMD" &> /dev/null; then
        echo -e "${RED}❌ 'migrate' command not found${NC}"
        echo -e "${YELLOW}💡 Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest${NC}"
        exit 1
    fi
}

# Show usage information
show_usage() {
    echo -e "${BLUE}📖 Database Migration Script${NC}"
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 up [N]           - Apply all or N up migrations"
    echo "  $0 down [N]         - Apply all or N down migrations"
    echo "  $0 goto V           - Migrate to version V"
    echo "  $0 force V          - Set version V but don't run migration"
    echo "  $0 drop             - Drop everything inside database"
    echo "  $0 create NAME      - Create new migration files"
    echo "  $0 version          - Print current migration version"
    echo ""
    echo -e "${YELLOW}Environment Variables:${NC}"
    echo "  DATABASE_URL        - Database connection string"
    echo "  MIGRATIONS_DIR      - Directory containing migration files (default: ./migrations)"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0 up               # Apply all pending migrations"
    echo "  $0 up 1             # Apply only 1 migration"
    echo "  $0 down 1           # Rollback 1 migration"
    echo "  $0 create add_users_table  # Create new migration"
}

# Apply migrations up
migrate_up() {
    local steps=${1:-""}
    
    echo -e "${BLUE}⬆️ Applying migrations up${NC}"
    
    if [ -n "$steps" ]; then
        echo -e "${YELLOW}📊 Applying $steps migration(s)${NC}"
        $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up "$steps"
    else
        echo -e "${YELLOW}📊 Applying all pending migrations${NC}"
        $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
    fi
    
    echo -e "${GREEN}✅ Migrations applied successfully${NC}"
}

# Apply migrations down
migrate_down() {
    local steps=${1:-""}
    
    echo -e "${RED}⬇️ Rolling back migrations${NC}"
    
    if [ -n "$steps" ]; then
        echo -e "${YELLOW}📊 Rolling back $steps migration(s)${NC}"
        read -p "Are you sure you want to rollback $steps migration(s)? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down "$steps"
            echo -e "${GREEN}✅ Rollback completed${NC}"
        else
            echo -e "${YELLOW}❌ Rollback cancelled${NC}"
            exit 0
        fi
    else
        echo -e "${RED}⚠️ This will rollback ALL migrations!${NC}"
        read -p "Are you sure you want to rollback ALL migrations? (type 'yes' to confirm): " -r
        if [[ $REPLY == "yes" ]]; then
            $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down
            echo -e "${GREEN}✅ All migrations rolled back${NC}"
        else
            echo -e "${YELLOW}❌ Rollback cancelled${NC}"
            exit 0
        fi
    fi
}

# Migrate to specific version
migrate_goto() {
    local version=$1
    
    if [ -z "$version" ]; then
        echo -e "${RED}❌ Version number required${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}🎯 Migrating to version $version${NC}"
    $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" goto "$version"
    echo -e "${GREEN}✅ Migrated to version $version${NC}"
}

# Force set version
migrate_force() {
    local version=$1
    
    if [ -z "$version" ]; then
        echo -e "${RED}❌ Version number required${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}⚠️ Force setting version to $version${NC}"
    read -p "Are you sure? This doesn't run migrations (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" force "$version"
        echo -e "${GREEN}✅ Version set to $version${NC}"
    else
        echo -e "${YELLOW}❌ Operation cancelled${NC}"
        exit 0
    fi
}

# Drop all database objects
migrate_drop() {
    echo -e "${RED}💥 This will DROP ALL database objects!${NC}"
    read -p "Are you absolutely sure? (type 'DROP ALL' to confirm): " -r
    if [[ $REPLY == "DROP ALL" ]]; then
        $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" drop
        echo -e "${RED}💥 All database objects dropped${NC}"
    else
        echo -e "${YELLOW}❌ Drop cancelled${NC}"
        exit 0
    fi
}

# Create new migration
create_migration() {
    local name=$1
    
    if [ -z "$name" ]; then
        echo -e "${RED}❌ Migration name required${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}📝 Creating new migration: $name${NC}"
    $MIGRATE_CMD create -ext sql -dir "$MIGRATIONS_DIR" -seq "$name"
    echo -e "${GREEN}✅ Migration files created in $MIGRATIONS_DIR${NC}"
    
    # List the created files
    ls -la "$MIGRATIONS_DIR"/*"$name"*
}

# Show current version
show_version() {
    echo -e "${BLUE}📊 Current migration version:${NC}"
    version=$($MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version 2>/dev/null || echo "No version information")
    echo -e "${GREEN}$version${NC}"
}

# Validate database connection
validate_connection() {
    echo -e "${YELLOW}🔍 Validating database connection...${NC}"
    if ! $MIGRATE_CMD -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version >/dev/null 2>&1; then
        echo -e "${RED}❌ Cannot connect to database${NC}"
        echo -e "${YELLOW}Database URL: ${DATABASE_URL}${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Database connection successful${NC}"
}

# Main script logic
main() {
    check_migrate_command
    
    local command=${1:-""}
    
    case $command in
        "up")
            validate_connection
            migrate_up "$2"
            ;;
        "down")
            validate_connection
            migrate_down "$2"
            ;;
        "goto")
            validate_connection
            migrate_goto "$2"
            ;;
        "force")
            validate_connection
            migrate_force "$2"
            ;;
        "drop")
            validate_connection
            migrate_drop
            ;;
        "create")
            create_migration "$2"
            ;;
        "version")
            validate_connection
            show_version
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
