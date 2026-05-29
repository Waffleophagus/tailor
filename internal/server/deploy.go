package server

import (
	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/deploy"
)

func toAPISetup(info *deploy.SetupInfo) *api.TailscaleSetupInfo {
	if info == nil {
		return nil
	}
	hints := make([]api.SetupHint, len(info.Hints))
	for i, hint := range info.Hints {
		hints[i] = api.SetupHint{ID: hint.ID, Message: hint.Message}
	}
	return &api.TailscaleSetupInfo{
		Required: info.Required,
		Hints:    hints,
	}
}

func (s *Server) attachSetup(status *api.LocalAPIStatusResponse, localAPIAvailable bool, deviceCount int) {
	status.Setup = toAPISetup(s.deploy.SetupInfo(localAPIAvailable, deviceCount))
}

func (s *Server) attachTopologySetup(response *api.TopologyResponse, localAPIAvailable bool) {
	response.Setup = toAPISetup(s.deploy.SetupInfo(localAPIAvailable, len(response.Devices)))
}
