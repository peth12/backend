#!/bin/bash

# Deployment script for SpendWise Pro Backend

set -e  # Exit on error

echo "================================"
echo "SpendWise Pro - Deployment Script"
echo "================================"
echo ""

# Build the application
echo "Step 1: Building application..."
./build.sh
echo ""

# Create deployment directory structure
echo "Step 2: Preparing deployment files..."
mkdir -p deploy
cp bin/server deploy/
cp -r uploads deploy/ 2>/dev/null || mkdir -p deploy/uploads

# Create a sample .env file for reference
cat > deploy/.env.example << 'EOF'
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=spendwise

# JWT Secret
JWT_SECRET=your_super_secret_jwt_key_change_this_in_production

# Server Configuration
PORT=8080

# SlipOK API (if using)
SLIPOK_API_KEY=your_slipok_api_key
EOF

echo ""
echo "✓ Deployment files ready in ./deploy directory"
echo ""
echo "Deployment checklist:"
echo "  1. Copy ./deploy directory to your server"
echo "  2. Create .env file on server (use .env.example as template)"
echo "  3. Ensure PostgreSQL is running"
echo "  4. Run: ./server"
echo ""
echo "Optional: Set up systemd service for automatic startup"
