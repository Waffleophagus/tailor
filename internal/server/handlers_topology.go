package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	build := "release"
	if devtailnet.Enabled {
		build = "dev"
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status:  "ok",
		Version: "dev",
		Build:   build,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if s.core.UseDevTailnet() {
		writeJSON(w, http.StatusOK, api.LocalAPIStatusResponse{
			Available:        true,
			LocalAPIEndpoint: "dev tailnet (" + devtailnet.Name + ")",
		})
		return
	}

	_, err := s.core.TopologyDevicesLogged(r.Context(), "status")
	if err != nil {
		status := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.core.LocalAPIEndpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&status, false, 0)
		writeJSON(w, http.StatusOK, status)
		return
	}

	status := api.LocalAPIStatusResponse{
		Available:        true,
		LocalAPIEndpoint: s.core.LocalAPIEndpoint(),
	}
	s.attachSetup(&status, true, 0)
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	devices, err := s.core.TopologyDevicesLogged(r.Context(), "topology")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, localapi.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		s.writeLocalAPIUnavailable(w, r, status, err, "topology unavailable")
		return
	}

	s.logger.Info("topology fetched",
		"device_count", len(devices),
		"request_id", RequestIDFromContext(r.Context()),
	)
	snapshot := s.core.TopologySnapshot(r.Context(), devices)
	s.attachTopologySetup(&snapshot, true)
	if s.core.CloudStatus().Authenticated {
		snapshot.StagedDrafts = s.core.StagedDrafts().Drafts
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleTopologySocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: topologyWebSocketOriginPatterns(r),
	})
	if err != nil {
		s.logger.Warn("topology websocket accept failed",
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	s.logger.Info("topology websocket connected",
		"remote", r.RemoteAddr,
		"request_id", RequestIDFromContext(r.Context()),
	)
	defer s.logger.Info("topology websocket disconnected",
		"remote", r.RemoteAddr,
		"request_id", RequestIDFromContext(r.Context()),
	)

	ctx := conn.CloseRead(r.Context())
	conn.SetReadLimit(64 << 10)

	var lastMessage []byte
	if err := s.writeTopologySocketMessage(ctx, conn, &lastMessage); err != nil {
		s.logger.Warn("topology websocket write failed",
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.writeTopologySocketMessage(ctx, conn, &lastMessage); err != nil {
				s.logger.Warn("topology websocket write failed",
					"error", err.Error(),
					"request_id", RequestIDFromContext(r.Context()),
				)
				return
			}
		}
	}
}

func (s *Server) writeTopologySocketMessage(ctx context.Context, conn *websocket.Conn, lastMessage *[]byte) error {
	message := s.topologySocketMessage(ctx)
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if bytes.Equal(encoded, *lastMessage) {
		return nil
	}
	*lastMessage = encoded

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return wsjson.Write(writeCtx, conn, message)
}

func (s *Server) topologySocketMessage(ctx context.Context) api.SocketMessage {
	devices, err := s.core.TopologyDevices(ctx)
	if err != nil {
		unavailable := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.core.LocalAPIEndpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&unavailable, false, 0)
		return api.SocketMessage{
			Type:    api.SocketMessageLocalAPIUnavailable,
			Payload: unavailable,
		}
	}

	snapshot := s.core.TopologySnapshot(ctx, devices)
	s.attachTopologySetup(&snapshot, true)
	if s.core.CloudStatus().Authenticated {
		snapshot.StagedDrafts = s.core.StagedDrafts().Drafts
	}
	return api.SocketMessage{
		Type:    api.SocketMessageTopologySnapshot,
		Payload: snapshot,
	}
}
