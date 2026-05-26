#!/usr/bin/env bash

cd "$(dirname "$0")" || exit 1

# Read the AI API key from the Windows user environment.
AIM_AI_API_KEY=$(powershell -Command "[Environment]::GetEnvironmentVariable('AIM_AI_API_KEY', 'User')" 2>/dev/null)

if [ -z "$AIM_AI_API_KEY" ]; then
    echo "[start] warning: AIM_AI_API_KEY was not found; AI features may be unavailable."
fi

export AIM_AI_API_KEY

is_aim_process() {
    local pid="$1"
    local process_name

    if [ -z "$pid" ]; then
        return 1
    fi

    process_name=$(MSYS_NO_PATHCONV=1 tasklist.exe /FI "PID eq $pid" /FO CSV /NH 2>/dev/null | head -n 1 | cut -d',' -f1 | tr -d '"')
    [ "$process_name" = "aim.exe" ]
}

wait_for_exit() {
    local pid="$1"
    local attempt

    for attempt in 1 2 3 4 5; do
        if ! is_aim_process "$pid"; then
            return 0
        fi
        sleep 1
    done

    return 1
}

if [ ! -x ./aim.exe ]; then
    echo "[start] error: ./aim.exe was not found or is not executable."
    exit 1
fi

# Stop the previous instance recorded by aim.pid before starting a new one.
if [ -f aim.pid ]; then
    OLD_PID=$(cat aim.pid)
    if [ -n "$OLD_PID" ]; then
        if is_aim_process "$OLD_PID"; then
            echo "[start] stopping previous AIM instance. PID=$OLD_PID"
            MSYS_NO_PATHCONV=1 taskkill.exe /PID "$OLD_PID" 2>/dev/null || true

            if ! wait_for_exit "$OLD_PID"; then
                echo "[start] previous instance did not exit cleanly; forcing stop. PID=$OLD_PID"
                MSYS_NO_PATHCONV=1 taskkill.exe /F /PID "$OLD_PID" 2>/dev/null || true
                sleep 1
            fi

            if is_aim_process "$OLD_PID"; then
                echo "[start] error: failed to stop previous AIM instance. PID=$OLD_PID"
                exit 1
            fi
        else
            echo "[start] ignoring stale aim.pid. PID=$OLD_PID"
        fi
    fi
    rm -f aim.pid
fi

mkdir -p logs

nohup ./aim.exe >> logs/aim.log 2>&1 &
NEW_PID=$!
echo "$NEW_PID" > aim.pid

sleep 2

if ! is_aim_process "$NEW_PID"; then
    rm -f aim.pid
    echo "[start] error: AIM failed to start. See logs/aim.log for details."
    tail -20 logs/aim.log 2>/dev/null
    exit 1
fi

echo "[start] AIM started. PID=$NEW_PID"
grep "\[ai\]\|\[server\]" logs/aim.log 2>/dev/null | tail -2
