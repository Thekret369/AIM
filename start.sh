#!/usr/bin/env bash

cd "$(dirname "$0")" || exit 1

# Read the AI API key from the Windows user environment.
LANLINE_AI_API_KEY=$(powershell -Command "[Environment]::GetEnvironmentVariable('LANLINE_AI_API_KEY', 'User')" 2>/dev/null)

if [ -z "$LANLINE_AI_API_KEY" ]; then
    echo "[start] warning: LANLINE_AI_API_KEY was not found; AI features may be unavailable."
fi

export LANLINE_AI_API_KEY

is_lanline_process() {
    local pid="$1"
    local process_name

    if [ -z "$pid" ]; then
        return 1
    fi

    process_name=$(MSYS_NO_PATHCONV=1 tasklist.exe /FI "PID eq $pid" /FO CSV /NH 2>/dev/null | head -n 1 | cut -d',' -f1 | tr -d '"')
    [ "$process_name" = "lanline.exe" ]
}

wait_for_exit() {
    local pid="$1"
    local attempt

    for attempt in 1 2 3 4 5; do
        if ! is_lanline_process "$pid"; then
            return 0
        fi
        sleep 1
    done

    return 1
}

if [ ! -x ./lanline.exe ]; then
    echo "[start] error: ./lanline.exe was not found or is not executable."
    exit 1
fi

# Stop the previous instance recorded by lanline.pid before starting a new one.
if [ -f lanline.pid ]; then
    OLD_PID=$(cat lanline.pid)
    if [ -n "$OLD_PID" ]; then
        if is_lanline_process "$OLD_PID"; then
            echo "[start] stopping previous LanLine instance. PID=$OLD_PID"
            MSYS_NO_PATHCONV=1 taskkill.exe /PID "$OLD_PID" 2>/dev/null || true

            if ! wait_for_exit "$OLD_PID"; then
                echo "[start] previous instance did not exit cleanly; forcing stop. PID=$OLD_PID"
                MSYS_NO_PATHCONV=1 taskkill.exe /F /PID "$OLD_PID" 2>/dev/null || true
                sleep 1
            fi

            if is_lanline_process "$OLD_PID"; then
                echo "[start] error: failed to stop previous LanLine instance. PID=$OLD_PID"
                exit 1
            fi
        else
            echo "[start] ignoring stale lanline.pid. PID=$OLD_PID"
        fi
    fi
    rm -f lanline.pid
fi

mkdir -p logs

nohup ./lanline.exe >> logs/lanline.log 2>&1 &
NEW_PID=$!
echo "$NEW_PID" > lanline.pid

sleep 2

if ! is_lanline_process "$NEW_PID"; then
    rm -f lanline.pid
    echo "[start] error: LanLine failed to start. See logs/lanline.log for details."
    tail -20 logs/lanline.log 2>/dev/null
    exit 1
fi

echo "[start] LanLine started. PID=$NEW_PID"
grep "\[ai\]\|\[server\]" logs/lanline.log 2>/dev/null | tail -2
