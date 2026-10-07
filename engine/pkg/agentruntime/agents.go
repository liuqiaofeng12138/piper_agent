package agentruntime

import "piper_go/pkg/distributor"

func (s *Service) HTTAgentID() string {
	if n := s.ctx.Node(); n != nil {
		return n.InstID + "-http-default"
	}
	return "local-http-default"
}

// EnsureHTTPAgent registers the default HttpAgent used by agent-driven runs.
func (s *Service) EnsureHTTPAgent() {
	id := s.HTTAgentID()
	if _, err := distributor.GetAgentByID(id); err == nil {
		return
	}
	inst := "local"
	if n := s.ctx.Node(); n != nil {
		inst = n.InstID
	}
	distributor.RegisterAgent(distributor.NewHttpAgentDoc(inst, "http-default"))
}
