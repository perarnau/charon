#!/bin/bash

# Charon Log Streaming Demo Runner
# This script helps you run the streaming demo easily

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEMO_BINARY="$SCRIPT_DIR/streaming_demo"
CHARON_URL="${CHARON_URL:-http://localhost:8080}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${GREEN}=================================${NC}"
    echo -e "${GREEN}  Charon Log Streaming Demo${NC}"
    echo -e "${GREEN}=================================${NC}"
    echo -e "Charon URL: ${BLUE}$CHARON_URL${NC}"
    echo ""
}

check_charon() {
    echo -e "${YELLOW}Checking Charon daemon...${NC}"
    if curl -s -f "$CHARON_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Charon daemon is running${NC}"
    else
        echo -e "${RED}❌ Charon daemon is not responding at $CHARON_URL${NC}"
        echo -e "${YELLOW}Please make sure charond is running first${NC}"
        exit 1
    fi
    echo ""
}

build_demo() {
    echo -e "${YELLOW}Building streaming demo...${NC}"
    cd "$SCRIPT_DIR"
    if go build -o streaming_demo main.go; then
        echo -e "${GREEN}✅ Demo built successfully${NC}"
    else
        echo -e "${RED}❌ Failed to build demo${NC}"
        exit 1
    fi
    echo ""
}

run_demo() {
    echo -e "${CYAN}Starting demo...${NC}"
    echo ""
    "$DEMO_BINARY" demo
}

submit_job() {
    local job_file="$1"
    if [[ ! -f "$job_file" ]]; then
        echo -e "${RED}❌ Job file not found: $job_file${NC}"
        exit 1
    fi
    
    echo -e "${CYAN}Submitting job: $job_file${NC}"
    echo ""
    "$DEMO_BINARY" submit "$job_file"
}

stream_logs() {
    local job_id="$1"
    if [[ -z "$job_id" ]]; then
        echo -e "${RED}❌ Job ID required${NC}"
        exit 1
    fi
    
    echo -e "${CYAN}Streaming logs for job: $job_id${NC}"
    echo ""
    "$DEMO_BINARY" stream "$job_id"
}

stream_events() {
    local job_id="$1"
    if [[ -z "$job_id" ]]; then
        echo -e "${RED}❌ Job ID required${NC}"
        exit 1
    fi
    
    echo -e "${CYAN}Streaming events for job: $job_id${NC}"
    echo ""
    "$DEMO_BINARY" events "$job_id"
}

show_usage() {
    echo -e "${BLUE}Usage: $0 [command] [args...]${NC}"
    echo ""
    echo -e "${YELLOW}Commands:${NC}"
    echo -e "  ${GREEN}demo${NC}                    - Run complete demo"
    echo -e "  ${GREEN}submit <job.yaml>${NC}      - Submit job and stream logs"
    echo -e "  ${GREEN}stream <job-id>${NC}        - Stream logs for existing job"
    echo -e "  ${GREEN}events <job-id>${NC}        - Stream events for existing job"
    echo -e "  ${GREEN}build${NC}                  - Build the demo binary"
    echo -e "  ${GREEN}check${NC}                  - Check Charon daemon status"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo -e "  $0 demo"
    echo -e "  $0 submit simple_job.yaml"
    echo -e "  $0 submit long_job.yaml"
    echo -e "  $0 stream abc123-def456-ghi789"
    echo -e "  $0 events abc123-def456-ghi789"
    echo ""
    echo -e "${YELLOW}Environment Variables:${NC}"
    echo -e "  ${GREEN}CHARON_URL${NC}    - Charon daemon URL (default: http://localhost:8080)"
    echo ""
    echo -e "${YELLOW}Available Job Files:${NC}"
    for yaml_file in "$SCRIPT_DIR"/*.yaml; do
        if [[ -f "$yaml_file" ]]; then
            basename "$yaml_file"
        fi
    done
}

# Main script logic
main() {
    print_header
    
    if [[ $# -eq 0 ]]; then
        show_usage
        exit 0
    fi
    
    local command="$1"
    shift
    
    case "$command" in
        demo)
            check_charon
            build_demo
            run_demo
            ;;
        submit)
            if [[ $# -eq 0 ]]; then
                echo -e "${RED}❌ Job file required${NC}"
                echo -e "Usage: $0 submit <job.yaml>"
                exit 1
            fi
            check_charon
            build_demo
            submit_job "$1"
            ;;
        stream)
            if [[ $# -eq 0 ]]; then
                echo -e "${RED}❌ Job ID required${NC}"
                echo -e "Usage: $0 stream <job-id>"
                exit 1
            fi
            check_charon
            build_demo
            stream_logs "$1"
            ;;
        events)
            if [[ $# -eq 0 ]]; then
                echo -e "${RED}❌ Job ID required${NC}"
                echo -e "Usage: $0 events <job-id>"
                exit 1
            fi
            check_charon
            build_demo
            stream_events "$1"
            ;;
        build)
            build_demo
            ;;
        check)
            check_charon
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            echo -e "${RED}❌ Unknown command: $command${NC}"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
