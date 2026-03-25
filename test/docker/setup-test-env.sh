#!/bin/bash
# LLM Gateway Black-box Test Environment Setup Script
# Usage: ./setup-test-env.sh [up|down|test|logs]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.test.yml"
GO_CMD="${GO_CMD:-go}"

case "${1:-up}" in
  up)
    echo "🚀 Starting LLM Gateway test environment..."
    docker compose -f "$COMPOSE_FILE" up --build -d
    echo ""
    echo "⏳ Waiting for services to be healthy..."
    sleep 5

    # Wait for postgres
    echo "Waiting for PostgreSQL..."
    until docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U llmgw -d llmgw_test; do
      sleep 1
    done
    echo "✅ PostgreSQL is ready"

    # Wait for llmgw server
    echo "Waiting for LLM Gateway server..."
    until curl -sf http://localhost:8080/auth/login > /dev/null 2>&1; do
      sleep 1
    done
    echo "✅ LLM Gateway server is ready"
    echo ""
    echo "📋 Test Environment Ready!"
    echo "   API Endpoint: http://localhost:8080"
    echo "   Database:     localhost:5433 (user: llmgw, db: llmgw_test)"
    echo ""
    echo "Run tests with:"
    echo "   ./setup-test-env.sh test"
    ;;

  down)
    echo "🛑 Stopping LLM Gateway test environment..."
    docker compose -f "$COMPOSE_FILE" down -v
    echo "✅ Environment stopped and volumes removed"
    ;;

  test)
    echo "🧪 Running black-box tests against test environment..."
    if ! command -v "$GO_CMD" >/dev/null 2>&1; then
      echo "❌ Go command not found: $GO_CMD"
      echo "   Set GO_CMD to your Go binary path, e.g. GO_CMD=/usr/local/bin/go"
      exit 1
    fi
    export TEST_API_BASE="http://localhost:8080"
    export TEST_DATABASE_URL="postgres://llmgw:llmgw_test_password@localhost:5433/llmgw_test?sslmode=disable"
    export TEST_JWT_SECRET="test-jwt-secret-for-blackbox-testing"
    cd "${SCRIPT_DIR}/../.."
    "$GO_CMD" test ./test/blackbox/... -v -timeout 60s
    ;;

  logs)
    docker compose -f "$COMPOSE_FILE" logs -f
    ;;

  status)
    echo "📊 Test Environment Status:"
    docker compose -f "$COMPOSE_FILE" ps
    echo ""
    echo "Health checks:"
    curl -sf http://localhost:8080/auth/login > /dev/null 2>&1 && echo "✅ API Server: OK" || echo "❌ API Server: DOWN"
    docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U llmgw -d llmgw_test > /dev/null 2>&1 && echo "✅ PostgreSQL: OK" || echo "❌ PostgreSQL: DOWN"
    ;;

  psql)
    docker compose -f "$COMPOSE_FILE" exec postgres psql -U llmgw -d llmgw_test
    ;;

  *)
    echo "Usage: $0 {up|down|test|logs|status|psql}"
    echo ""
    echo "Commands:"
    echo "  up     - Start the test environment (PostgreSQL + LLM Gateway)"
    echo "  down   - Stop and remove the test environment"
    echo "  test   - Run black-box tests against the running environment"
    echo "  logs   - Show container logs"
    echo "  status - Show environment status"
    echo "  psql   - Connect to PostgreSQL"
    exit 1
    ;;
esac