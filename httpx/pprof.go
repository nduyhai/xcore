package httpx

import "github.com/gin-contrib/pprof"

func (s *Server) initPprof() error {
	if s.cfg.Pprof.Enabled {
		pprof.Register(s.engine, s.cfg.Pprof.Prefix)
	}
	return nil
}
