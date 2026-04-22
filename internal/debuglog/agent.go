package debuglog

import (
	"encoding/json"
	"os"
	"time"
)

// AgentLog appends one NDJSON line for Cursor debug sessions (session 6f2144).
func AgentLog(hypothesisID, location, message string, data map[string]any) {
	const path = "/Users/mohayat/projects/KH/termite/.cursor/debug-6f2144.log"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	payload := map[string]any{
		"sessionId":    "6f2144",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"timestamp":    time.Now().UnixMilli(),
	}
	if len(data) > 0 {
		payload["data"] = data
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
}
