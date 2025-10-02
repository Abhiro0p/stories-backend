#!/bin/bash

# Stories Backend Database Seeding Script
# This script creates sample data for development and testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DATABASE_URL=${DATABASE_URL:-"postgresql://stories_user:stories_pass@localhost:5432/stories_db?sslmode=disable"}
SEED_USERS=${SEED_USERS:-10}
SEED_STORIES=${SEED_STORIES:-50}
SEED_FOLLOWS=${SEED_FOLLOWS:-25}

echo -e "${BLUE}🌱 Starting database seeding...${NC}"

# Check if database is accessible
echo -e "${YELLOW}📡 Checking database connection...${NC}"
if ! psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${RED}❌ Cannot connect to database${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Database connection successful${NC}"

# Create seed data SQL
cat << 'EOF' > /tmp/seed.sql
-- Seed data for Stories Backend

-- Sample users
INSERT INTO users (id, email, username, password_hash, full_name, bio, is_active, is_verified, created_at) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'alice@example.com', 'alice_stories', '$2a$12$example_hash_1', 'Alice Johnson', 'Travel enthusiast and photographer', true, true, NOW() - INTERVAL '30 days'),
('550e8400-e29b-41d4-a716-446655440002', 'bob@example.com', 'bob_adventures', '$2a$12$example_hash_2', 'Bob Smith', 'Adventure seeker and storyteller', true, false, NOW() - INTERVAL '25 days'),
('550e8400-e29b-41d4-a716-446655440003', 'carol@example.com', 'carol_creative', '$2a$12$example_hash_3', 'Carol Davis', 'Creative writer and artist', true, true, NOW() - INTERVAL '20 days'),
('550e8400-e29b-41d4-a716-446655440004', 'david@example.com', 'david_tech', '$2a$12$example_hash_4', 'David Wilson', 'Tech enthusiast and developer', true, false, NOW() - INTERVAL '15 days'),
('550e8400-e29b-41d4-a716-446655440005', 'emma@example.com', 'emma_lifestyle', '$2a$12$example_hash_5', 'Emma Brown', 'Lifestyle blogger and influencer', true, true, NOW() - INTERVAL '10 days');

-- Sample stories
INSERT INTO stories (id, author_id, type, text, visibility, view_count, expires_at, created_at) VALUES
('660e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', 'text', 'Just discovered this amazing coffee shop in downtown! ☕️ The atmosphere is perfect for working.', 'public', 15, NOW() + INTERVAL '20 hours', NOW() - INTERVAL '4 hours'),
('660e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440002', 'text', 'Hiking trip to the mountains was incredible! The view from the top was absolutely breathtaking 🏔️', 'public', 23, NOW() + INTERVAL '18 hours', NOW() - INTERVAL '6 hours'),
('660e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440003', 'text', 'Working on a new art piece today. Sometimes inspiration strikes when you least expect it! 🎨', 'public', 8, NOW() + INTERVAL '16 hours', NOW() - INTERVAL '8 hours'),
('660e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440004', 'text', 'Just deployed a new feature to production. The feeling of seeing your code come to life! 💻', 'public', 12, NOW() + INTERVAL '14 hours', NOW() - INTERVAL '10 hours'),
('660e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440005', 'text', 'Morning yoga session complete ✨ Starting the day with mindfulness and positive energy!', 'public', 19, NOW() + INTERVAL '12 hours', NOW() - INTERVAL '12 hours');

-- Sample follow relationships
INSERT INTO follows (id, follower_id, followee_id, created_at) VALUES
('770e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440002', NOW() - INTERVAL '5 days'),
('770e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440003', NOW() - INTERVAL '4 days'),
('770e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', NOW() - INTERVAL '3 days'),
('770e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440004', NOW() - INTERVAL '2 days'),
('770e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440005', NOW() - INTERVAL '1 day');

-- Sample reactions
INSERT INTO reactions (id, story_id, user_id, type, created_at) VALUES
('880e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440002', 'like', NOW() - INTERVAL '3 hours'),
('880e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440003', 'love', NOW() - INTERVAL '2 hours'),
('880e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', 'fire', NOW() - INTERVAL '5 hours'),
('880e8400-e29b-41d4-a716-446655440004', '660e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440004', 'wow', NOW() - INTERVAL '7 hours');

-- Sample story views
INSERT INTO story_views (id, story_id, viewer_id, viewed_at, ip_address) VALUES
('990e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440002', NOW() - INTERVAL '3 hours', '192.168.1.100'),
('990e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440003', NOW() - INTERVAL '2 hours', '192.168.1.101'),
('990e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', NOW() - INTERVAL '5 hours', '192.168.1.102');

-- Update user stats
UPDATE users SET 
    follower_count = (SELECT COUNT(*) FROM follows WHERE followee_id = users.id),
    following_count = (SELECT COUNT(*) FROM follows WHERE follower_id = users.id),
    story_count = (SELECT COUNT(*) FROM stories WHERE author_id = users.id AND deleted_at IS NULL);

-- Update story reaction counts
UPDATE stories SET 
    reaction_count = (SELECT COUNT(*) FROM reactions WHERE story_id = stories.id);

EOF

# Execute seed data
echo -e "${YELLOW}🌱 Inserting seed data...${NC}"
if psql "$DATABASE_URL" -f /tmp/seed.sql > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Seed data inserted successfully${NC}"
else
    echo -e "${RED}❌ Failed to insert seed data${NC}"
    exit 1
fi

# Cleanup
rm -f /tmp/seed.sql

# Summary
echo -e "${BLUE}📊 Seed Summary:${NC}"
echo -e "  Users created: 5"
echo -e "  Stories created: 5"
echo -e "  Follow relationships: 5"
echo -e "  Reactions: 4"
echo -e "  Story views: 3"

echo -e "${GREEN}🎉 Database seeding completed successfully!${NC}"

# Display sample users for testing
echo -e "${BLUE}👥 Sample users for testing:${NC}"
echo -e "  alice@example.com / alice_stories"
echo -e "  bob@example.com / bob_adventures"
echo -e "  carol@example.com / carol_creative"
echo -e "  david@example.com / david_tech"
echo -e "  emma@example.com / emma_lifestyle"
echo -e "${YELLOW}💡 All users have the password 'password123' (hashed)${NC}"
