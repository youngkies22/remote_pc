package agent

import (
	"context"

	"go.uber.org/zap"

	"remote_pc/internal/agent/fsops"
	"remote_pc/internal/agent/input"
	"remote_pc/internal/agent/procs"
	"remote_pc/internal/agent/screen"
	"remote_pc/internal/agent/sysinfo"
	"remote_pc/internal/agent/winservices"
	"remote_pc/internal/protocol"
)

// dispatch menangani pesan dari server. Perintah request/response dijalankan di
// goroutine terpisah agar tidak memblokir loop pembaca; event input & kontrol
// stream dieksekusi langsung.
func (s *session) dispatch(ctx context.Context, env *protocol.Envelope) {
	switch env.Type {
	case protocol.TypePing:
		if pong, err := env.Reply(protocol.TypePong, nil); err == nil {
			s.enqueue(ctx, pong)
		}

	// --- Request/response ---
	case protocol.TypeSysInfo:
		s.respond(env, func() (interface{}, error) { return sysinfo.FullInfo() })
	case protocol.TypeScreenshot:
		s.respond(env, func() (interface{}, error) { return screen.Capture() })
	case protocol.TypeProcList:
		s.respond(env, func() (interface{}, error) { return procs.List() })
	case protocol.TypeProcKill:
		s.respond(env, func() (interface{}, error) { return okOrErr(killProc(env)) })
	case protocol.TypeSvcList:
		s.respond(env, func() (interface{}, error) { return winservices.List() })
	case protocol.TypeSvcControl:
		s.respond(env, func() (interface{}, error) { return okOrErr(controlSvc(env)) })
	case protocol.TypeFSDrives:
		s.respond(env, func() (interface{}, error) { return fsops.Drives(), nil })
	case protocol.TypeFSList:
		s.respond(env, func() (interface{}, error) { return fsList(env) })
	case protocol.TypeFSRead:
		s.respond(env, func() (interface{}, error) { return fsRead(env) })
	case protocol.TypeFSWrite:
		s.respond(env, func() (interface{}, error) { return okOrErr(fsWrite(env)) })
	case protocol.TypeFSMkdir:
		s.respond(env, func() (interface{}, error) { return okOrErr(fsPathOp(env, fsops.Mkdir)) })
	case protocol.TypeFSDelete:
		s.respond(env, func() (interface{}, error) { return okOrErr(fsPathOp(env, fsops.Delete)) })
	case protocol.TypeFSRename:
		s.respond(env, func() (interface{}, error) { return okOrErr(fsTwoPathOp(env, fsops.Rename)) })
	case protocol.TypeFSCopy:
		s.respond(env, func() (interface{}, error) { return okOrErr(fsTwoPathOp(env, fsops.Copy)) })
	case protocol.TypeFSMove:
		s.respond(env, func() (interface{}, error) { return okOrErr(fsTwoPathOp(env, fsops.Move)) })

	// --- Streaming & input ---
	case protocol.TypeScreenStart:
		s.startScreen()
	case protocol.TypeScreenStop:
		s.stopScreen()
	case protocol.TypeScreenQuality:
		var req protocol.ScreenQualityRequest
		if env.Decode(&req) == nil {
			s.setScreenQuality(req.Quality)
		}
	case protocol.TypeInputMouse:
		var ev protocol.MouseEvent
		if env.Decode(&ev) == nil {
			input.Mouse(ev)
		}
	case protocol.TypeInputKey:
		var ev protocol.KeyEvent
		if env.Decode(&ev) == nil {
			input.Key(ev)
		}
	case protocol.TypeTermStart:
		var req protocol.TermStart
		_ = env.Decode(&req)
		s.startTerminal(req.Shell)
	case protocol.TypeTermInput:
		var req protocol.TermData
		if env.Decode(&req) == nil {
			s.termInput(req.Data)
		}
	case protocol.TypeTermStop:
		s.stopTerminal()

	// --- Power control ---
	case protocol.TypePowerShutdown:
		s.powerControl(false)
	case protocol.TypePowerRestart:
		s.powerControl(true)

	// --- Pesan ke client ---
	case protocol.TypeMessage:
		s.showMessage(env)

	default:
		s.log.Debug("pesan server tak dikenal", zap.String("type", string(env.Type)))
	}
}

// respond menjalankan fn di goroutine dan mengirim hasilnya sebagai response/error.
func (s *session) respond(env *protocol.Envelope, fn func() (interface{}, error)) {
	go func() {
		payload, err := fn()
		if err != nil {
			s.enqueue(s.ctx, env.ErrorReply(err.Error()))
			return
		}
		reply, rerr := env.Reply(protocol.TypeResponse, payload)
		if rerr != nil {
			s.enqueue(s.ctx, env.ErrorReply(rerr.Error()))
			return
		}
		s.enqueue(s.ctx, reply)
	}()
}

// statusOK adalah payload standar untuk operasi tanpa data balik.
type statusOK struct {
	Status string `json:"status"`
}

func okOrErr(err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	return statusOK{Status: "ok"}, nil
}

func killProc(env *protocol.Envelope) error {
	var req protocol.ProcKillRequest
	if err := env.Decode(&req); err != nil {
		return err
	}
	return procs.Kill(req.PID)
}

func controlSvc(env *protocol.Envelope) error {
	var req protocol.SvcControlRequest
	if err := env.Decode(&req); err != nil {
		return err
	}
	return winservices.Control(req.Name, req.Action)
}

func fsList(env *protocol.Envelope) (interface{}, error) {
	var req protocol.FSPathRequest
	if err := env.Decode(&req); err != nil {
		return nil, err
	}
	return fsops.List(req.Path)
}

func fsRead(env *protocol.Envelope) (interface{}, error) {
	var req protocol.FSPathRequest
	if err := env.Decode(&req); err != nil {
		return nil, err
	}
	return fsops.Read(req.Path)
}

func fsWrite(env *protocol.Envelope) error {
	var req protocol.FSWriteRequest
	if err := env.Decode(&req); err != nil {
		return err
	}
	return fsops.Write(req.Path, req.Data)
}

func fsPathOp(env *protocol.Envelope, op func(string) error) error {
	var req protocol.FSPathRequest
	if err := env.Decode(&req); err != nil {
		return err
	}
	return op(req.Path)
}

func fsTwoPathOp(env *protocol.Envelope, op func(string, string) error) error {
	var req protocol.FSTwoPathRequest
	if err := env.Decode(&req); err != nil {
		return err
	}
	return op(req.Src, req.Dst)
}
