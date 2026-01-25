#!/bin/bash
set -euo pipefail

# ratelord orchestration loop
# Improves upon original by adding iteration limits, structured logging,
# and explicitly referencing the sentinel-checked LOOP_PROMPT.md.

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <goal_description>"
  exit 1
fi

CURENNT_GOAL=$1

MAX_ITERATIONS=50
LOG_DIR="logs"
PROMPT_FILE="LOOP_PROMPT.md"
OPENCODE_BIN=$(command -v opencode || echo "opencode")
OPENCODE_AGENT="orchestrator"
OPENCODE_GOAL="Follow execution protocol $PROMPT_FILE to achieve $CURENNT_GOAL. At the end of every iteration, you MUST use the terminal agent to commit all changes."
OPENCODE_COMMAND="run --agent $OPENCODE_AGENT $OPENCODE_GOAL -f $PROMPT_FILE"

# Colors for TUI-lite experience
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

mkdir -p "$LOG_DIR"

# Pre-flight checks
if ! command -v opencode &> /dev/null; then
  echo -e "${RED}❌ Error: 'opencode' command not found.${NC}"
  exit 1
fi

if [[ ! -f "$PROMPT_FILE" ]]; then
  echo -e "${RED}❌ Error: Prompt file '$PROMPT_FILE' not found.${NC}"
  exit 1
fi

echo -e "${BLUE}🔄 Starting ratelord loop (max iterations: $MAX_ITERATIONS)${NC}"
echo -e "${BLUE}📝 Prompt: $PROMPT_FILE${NC}"
echo -e "${BLUE}📂 Logs: $LOG_DIR/${NC}"

# Cleanup on interrupt
trap 'echo -e "\n${YELLOW}🛑 Loop interrupted by user. Cleaning up...${NC}"; exit 1' SIGINT

START_TIME=$(date +%s)

for ((i=1; i<=MAX_ITERATIONS; i++)); do
  ITER_START=$(date +%s)
  TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
  LOG_FILE="$LOG_DIR/iteration_${i}_${TIMESTAMP}.log"

  echo -e "\n${YELLOW}--- Iteration $i / $MAX_ITERATIONS ---${NC}" | tee -a "$LOG_FILE"

  # Run opencode with the orchestrator agent.
  # Run in background to monitor for inactivity (5 minute timeout).
  ($OPENCODE_BIN $OPENCODE_COMMAND 2>&1 | tee -a "$LOG_FILE" 2>&1) &
  PID=$!

  TIMEOUT=300
  KILLED=0

  while kill -0 $PID 2>/dev/null; do
    sleep 5
    if [[ "$OSTYPE" == "darwin"* ]]; then
      LAST_MOD=$(stat -f %m "$LOG_FILE")
    else
      LAST_MOD=$(stat -c %Y "$LOG_FILE")
    fi
    NOW=$(date +%s)

    if (( NOW - LAST_MOD > TIMEOUT )); then
      echo -e "\n${RED}❌ Timeout: No log output for ${TIMEOUT}s. Killing process...${NC}" | tee -a "$LOG_FILE"
      kill -9 $PID 2>/dev/null || true
      KILLED=1
      break
    fi
  done

  set +e
  wait $PID
  EXIT_CODE=$?
  set -e

  if [ $KILLED -eq 1 ]; then
    echo -e "${RED}❌ Error: Iteration timed out.${NC}" | tee -a "$LOG_FILE"
    exit 1
  fi

  if [ $EXIT_CODE -eq 0 ]; then
    # Check if the agent signalled completion
    if grep -q "<promise>DONE</promise>" "$LOG_FILE"; then
      echo -e "${GREEN}✅ Success: ALL TASKS DONE signal received.${NC}" | tee -a "$LOG_FILE"
      break
    elif grep -q "<promise>NEXT_TASK</promise>" "$LOG_FILE"; then
      echo -e "${BLUE}⏭️ Task complete. NEXT_TASK signal received. Continuing...${NC}" | tee -a "$LOG_FILE"
    else
      echo -e "${YELLOW}⚠️ Iteration finished without explicit signal. Continuing...${NC}" | tee -a "$LOG_FILE"
    fi
  else
    echo -e "${RED}❌ Error: opencode failed with exit code $EXIT_CODE. Check $LOG_FILE${NC}" | tee -a "$LOG_FILE"
    exit 1
  fi

  # Safety check: if the log file is just the header we added
  if [[ $(wc -l < "$LOG_FILE") -le 1 ]]; then
    echo -e "${RED}❌ Error: Empty log file (or only header). opencode may have crashed.${NC}" | tee -a "$LOG_FILE"
    exit 1
  fi

  ITER_END=$(date +%s)
  DURATION=$((ITER_END - ITER_START))
  echo -e "${GREEN}Iteration $i complete (${DURATION}s). Continuing...${NC}"
done

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

echo -e "\n${BLUE}🏁 Loop finished in ${TOTAL_DURATION}s.${NC}"
if (( i > MAX_ITERATIONS )); then
  echo -e "${YELLOW}🛑 Reached maximum iterations ($MAX_ITERATIONS).${NC}" | tee -a "$LOG_DIR/final_status.log"
else
  echo -e "${GREEN}✅ Project signaled completion or was stopped manually.${NC}"
fi
