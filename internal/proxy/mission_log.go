package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	contracts "github.com/roykaushik07/kestrel-contracts/go"
)

// logMissionStatus decodes single-mission responses from dispatch into the
// shared contract type and logs the mission status, leaving the body intact.
func logMissionStatus(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Request.URL.Path, "/v1/missions/") ||
		!strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	var m contracts.Mission
	if err := json.Unmarshal(body, &m); err == nil && m.ID != "" {
		slog.Info("mission fetched", "mission", m.ID, "status", m.Status, "priority", m.Priority)
	}
	return nil
}
