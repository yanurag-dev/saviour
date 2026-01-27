#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Running Saviour Test Suite${NC}"
echo "=================================="

# Run tests with coverage
echo -e "\n${YELLOW}Running tests...${NC}"
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Generate coverage report
echo -e "\n${YELLOW}Generating coverage report...${NC}"
go tool cover -func=coverage.out

# Calculate total coverage
TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "\n${GREEN}Total Coverage: ${TOTAL_COVERAGE}${NC}"

# Generate HTML coverage report
echo -e "\n${YELLOW}Generating HTML coverage report...${NC}"
go tool cover -html=coverage.out -o coverage.html
echo -e "${GREEN}HTML coverage report generated: coverage.html${NC}"

# Check coverage threshold
COVERAGE=$(echo $TOTAL_COVERAGE | sed 's/%//')
THRESHOLD=35.0

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo -e "${RED}❌ Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%${NC}"
    exit 1
else
    echo -e "${GREEN}✅ Coverage ${COVERAGE}% meets threshold ${THRESHOLD}%${NC}"
fi

echo -e "\n${GREEN}All tests passed!${NC}"
