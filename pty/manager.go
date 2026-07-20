// Package pty provides a PTY-based session manager. Each session runs a
// command in a pseudo-terminal with buffered output, interactive input, and
// lifecycle management. The API mirrors tmux semantics: Create, List, Peek,
// Write, Kill, Wait.
package pty

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

type State string

const (
	StateRunning   State = "running"
	StateCompleted State = "completed"
	StateKilled    State = "killed"
	StateFailed    State = "failed"
)

const (
	DefaultTimeout = 30 * time.Minute
	killGrace      = 5 * time.Second
	shutdownGrace  = 2 * time.Second
)

type Info struct {
	ID             string    `json:"id"`
	Kind           string    `json:"kind,omitempty"`
	Name           string    `json:"name,omitempty"`
	Command        string    `json:"command"`
	PID            int       `json:"pid"`
	StartedAt      time.Time `json:"started_at"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	EndedAt        time.Time `json:"ended_at,omitempty"`
	ActivitySeq    int64     `json:"activity_seq,omitempty"`
	OutputBytes    int64     `json:"output_bytes,omitempty"`
	ExitCode       int       `json:"exit_code"`
	State          State     `json:"state"`
	KillCause      string    `json:"kill_cause,omitempty"`
}

type EventAction string

const (
	EventSessionCreated EventAction = "created"
	EventSessionUpdated EventAction = "updated"
	EventSessionOutput  EventAction = "output"
	EventSessionClosed  EventAction = "closed"
)

type Event struct {
	Action      EventAction `json:"action"`
	Info        Info        `json:"info"`
	OutputBytes int         `json:"output_bytes,omitempty"`
}

type session struct {
	Info
	cmd        *exec.Cmd // nil for func sessions and platform-managed PTY sessions
	output     *OutputBuffer
	pty        *ptyHandle      // nil for func sessions
	input      *io.PipeWriter  // non-nil for input-capable in-process sessions
	pumpDone   <-chan struct{} // nil for func sessions
	done       chan struct{}
	peekOff    int64
	cancel     context.CancelFunc // non-nil for func sessions
	closeInput func(error)
}

// RunOpts controls how a session is created.
type RunOpts struct {
	Name    string
	Timeout time.Duration
	Env     []string
	Ctx     context.Context
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	onDone   func(Info)
	onEvent  func(Event)
	bufCap   int
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*session)}
}

func (m *Manager) SetOnEvent(fn func(Event)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onEvent = fn
}

func (m *Manager) SetOnDone(fn func(Info)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onDone = fn
}

// Create starts a shell command in a PTY session.
func (m *Manager) Create(workDir, cmdLine, name string, timeout time.Duration, env []string, outputFile string) (Info, error) {
	return m.create(workDir, cmdLine, name, timeout, env, outputFile, true)
}

// CreateRaw starts a shell command in a PTY session and preserves terminal
// control bytes in captured output. Use this for remote terminal attach.
func (m *Manager) CreateRaw(workDir, cmdLine, name string, timeout time.Duration, env []string, outputFile string) (Info, error) {
	return m.create(workDir, cmdLine, name, timeout, env, outputFile, false)
}

func (m *Manager) create(workDir, cmdLine, name string, timeout time.Duration, env []string, outputFile string, stripANSI bool) (Info, error) {
	if strings.TrimSpace(cmdLine) == "" {
		return Info{}, errors.New("empty command")
	}
	c := ShellCommand(cmdLine)
	c.Dir = workDir
	if len(env) > 0 {
		c.Env = mergeEnv(os.Environ(), env)
	}
	return m.start(c, cmdLine, name, timeout, outputFile, stripANSI)
}

// CreateFunc starts a goroutine-based session. The function fn runs in a
// goroutine; its output (written to w) is captured in the same OutputBuffer
// used by PTY sessions, so Peek/Kill/Wait/Done work identically.
// The session context is derived from parentCtx (or context.Background if nil).
func (m *Manager) CreateFunc(parentCtx context.Context, name string, timeout time.Duration, fn func(ctx context.Context, w io.Writer) error) (Info, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if name == "" {
		name = "func"
	}

	id, err := genID()
	if err != nil {
		return Info{}, err
	}

	buf, err := m.newBuffer("", true)
	if err != nil {
		return Info{}, err
	}

	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, timeout)

	startedAt := time.Now()
	info := Info{
		ID:             id,
		Kind:           "task",
		Name:           name,
		Command:        name,
		StartedAt:      startedAt,
		LastActivityAt: startedAt,
		ActivitySeq:    1,
		State:          StateRunning,
	}
	s := &session{Info: info, output: buf, done: make(chan struct{}), cancel: cancel}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	m.watchOutput(s)
	m.emit(Event{Action: EventSessionCreated, Info: info})

	go m.superviseFunc(s, ctx, fn)
	return info, nil
}

// CreateInteractiveFunc starts an in-process session that accepts input through
// Manager.Write and captures output in the same buffer used by PTY sessions.
func (m *Manager) CreateInteractiveFunc(parentCtx context.Context, name, command string, timeout time.Duration, stripANSI bool, fn func(ctx context.Context, r io.Reader, w io.Writer) error) (Info, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if name == "" {
		name = "interactive"
	}
	if command == "" {
		command = name
	}

	id, err := genID()
	if err != nil {
		return Info{}, err
	}

	buf, err := m.newBuffer("", stripANSI)
	if err != nil {
		return Info{}, err
	}

	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	pr, pw := io.Pipe()

	startedAt := time.Now()
	info := Info{
		ID:             id,
		Kind:           "task",
		Name:           name,
		Command:        command,
		StartedAt:      startedAt,
		LastActivityAt: startedAt,
		ActivitySeq:    1,
		State:          StateRunning,
	}
	s := &session{
		Info:   info,
		output: buf,
		input:  pw,
		done:   make(chan struct{}),
		cancel: cancel,
		closeInput: func(err error) {
			_ = pr.CloseWithError(err)
			_ = pw.CloseWithError(err)
		},
	}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	m.watchOutput(s)
	m.emit(Event{Action: EventSessionCreated, Info: info})

	go m.superviseInteractiveFunc(s, ctx, pr, fn)
	return info, nil
}

func (m *Manager) superviseFunc(s *session, ctx context.Context, fn func(context.Context, io.Writer) error) {
	defer s.cancel()

	var fnErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				fnErr = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		fnErr = fn(ctx, s.output)
	}()

	m.finishSession(s, fnErr, false)
}

func (m *Manager) superviseInteractiveFunc(s *session, ctx context.Context, r io.Reader, fn func(context.Context, io.Reader, io.Writer) error) {
	defer s.cancel()

	if closer, ok := r.(interface{ CloseWithError(error) error }); ok {
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				_ = closer.CloseWithError(ctx.Err())
			case <-done:
			}
		}()
		defer close(done)
	}

	var fnErr error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				fnErr = fmt.Errorf("panic: %v\n%s", recovered, debug.Stack())
			}
		}()
		fnErr = fn(ctx, r, s.output)
	}()

	if s.closeInput != nil {
		s.closeInput(io.EOF)
	}

	m.finishSession(s, fnErr, true)
}

func (m *Manager) finishSession(s *session, fnErr error, eofIsSuccess bool) {
	state, exitCode := StateCompleted, 0
	killCause := ""

	if fnErr != nil {
		switch {
		case errors.Is(fnErr, context.DeadlineExceeded):
			state = StateKilled
			killCause = "timeout"
		case errors.Is(fnErr, context.Canceled):
			state = StateKilled
			killCause = m.getKillCause(s)
			if killCause == "" {
				killCause = "canceled"
			}
		case eofIsSuccess && errors.Is(fnErr, io.EOF):
			// clean exit
		default:
			state = StateFailed
			exitCode = 1
			s.output.AppendError(fnErr.Error())
		}
	}

	m.mu.Lock()
	s.EndedAt = time.Now()
	s.LastActivityAt = s.EndedAt
	s.ActivitySeq++
	s.ExitCode = exitCode
	s.State = state
	s.KillCause = killCause
	infoCopy := s.Info
	m.mu.Unlock()

	s.output.Close()
	close(s.done)
	m.emit(Event{Action: EventSessionClosed, Info: infoCopy})

	m.mu.Lock()
	onDone := m.onDone
	m.mu.Unlock()
	if onDone != nil {
		defer func() { _ = recover() }()
		onDone(infoCopy)
	}
}

// CreateCmd starts a command with explicit binary and args in a PTY session.
func (m *Manager) CreateCmd(workDir, binary string, args []string, name string, timeout time.Duration, env []string, outputFile string) (Info, error) {
	return m.createCmd(workDir, binary, args, name, timeout, env, outputFile, true)
}

// CreateCmdRaw starts a command with explicit binary and args in a PTY session
// and preserves terminal control bytes in captured output.
func (m *Manager) CreateCmdRaw(workDir, binary string, args []string, name string, timeout time.Duration, env []string, outputFile string) (Info, error) {
	return m.createCmd(workDir, binary, args, name, timeout, env, outputFile, false)
}

func (m *Manager) createCmd(workDir, binary string, args []string, name string, timeout time.Duration, env []string, outputFile string, stripANSI bool) (Info, error) {
	c := exec.Command(binary, args...)
	c.Dir = workDir
	if len(env) > 0 {
		c.Env = mergeEnv(os.Environ(), env)
	}
	display := binary + " " + strings.Join(args, " ")
	return m.start(c, display, name, timeout, outputFile, stripANSI)
}

// mergeEnv merges override env vars into base, replacing any existing keys.
func mergeEnv(base, override []string) []string {
	overrideKeys := make(map[string]bool, len(override))
	for _, e := range override {
		if k, _, ok := strings.Cut(e, "="); ok {
			overrideKeys[k] = true
		}
	}
	result := make([]string, 0, len(base)+len(override))
	for _, e := range base {
		if k, _, ok := strings.Cut(e, "="); ok && overrideKeys[k] {
			continue
		}
		result = append(result, e)
	}
	return append(result, override...)
}

func (m *Manager) start(c *exec.Cmd, cmdDisplay, name string, timeout time.Duration, outputFile string, stripANSI bool) (Info, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if name == "" {
		name = labelFromCommand(cmdDisplay)
	}

	id, err := genID()
	if err != nil {
		return Info{}, err
	}

	buf, err := m.newBuffer(outputFile, stripANSI)
	if err != nil {
		return Info{}, err
	}

	p, err := startPTY(c)
	if err != nil {
		buf.Close()
		return Info{}, fmt.Errorf("start pty: %w", err)
	}

	startedAt := time.Now()
	info := Info{
		ID:             id,
		Kind:           "task",
		Name:           name,
		Command:        cmdDisplay,
		PID:            p.PID(),
		StartedAt:      startedAt,
		LastActivityAt: startedAt,
		ActivitySeq:    1,
		State:          StateRunning,
	}
	s := &session{Info: info, cmd: c, output: buf, pty: p, done: make(chan struct{})}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	m.watchOutput(s)
	m.emit(Event{Action: EventSessionCreated, Info: info})

	s.pumpDone = pumpOutput(p, buf)

	go m.supervise(s, timeout)
	return info, nil
}

func (m *Manager) supervise(s *session, timeout time.Duration) {
	waitDone := make(chan error, 1)
	processDone := make(chan struct{})
	go func() {
		waitDone <- s.pty.Wait()
		close(processDone)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var (
		waitErr   error
		killCause string
	)

	select {
	case err := <-waitDone:
		waitErr = err
	case <-timer.C:
		killCause = fmt.Sprintf("timeout after %s", timeout)
		m.setKillCause(s, killCause)
		m.forceKill(s, processDone)
		waitErr = <-waitDone
	}

	if recorded := m.getKillCause(s); recorded != "" {
		killCause = recorded
	}

	state, exitCode := StateCompleted, 0
	if waitErr != nil {
		var exitErr interface{ ExitCode() int }
		switch {
		case errors.As(waitErr, &exitErr):
			exitCode = exitErr.ExitCode()
			if killCause != "" {
				state = StateKilled
			} else {
				state = StateFailed
			}
		default:
			exitCode = -1
			state = StateFailed
		}
	} else if killCause != "" {
		state = StateKilled
	}

	if s.pty != nil {
		s.pty.Close()
	}
	if s.pumpDone != nil {
		<-s.pumpDone
	}

	m.mu.Lock()
	s.EndedAt = time.Now()
	s.LastActivityAt = s.EndedAt
	s.ActivitySeq++
	s.ExitCode = exitCode
	s.State = state
	s.KillCause = killCause
	infoCopy := s.Info
	m.mu.Unlock()

	s.output.Close()
	close(s.done)
	m.emit(Event{Action: EventSessionClosed, Info: infoCopy})

	m.mu.Lock()
	fn := m.onDone
	m.mu.Unlock()
	if fn != nil {
		defer func() { _ = recover() }()
		fn(infoCopy)
	}
}

func (m *Manager) forceKill(s *session, done <-chan struct{}) {
	if s.pty == nil {
		return
	}
	_ = s.pty.Signal(false)
	timer := time.NewTimer(killGrace)
	defer timer.Stop()
	select {
	case <-done:
		return
	case <-timer.C:
	}
	_ = s.pty.Signal(true)
}

func (m *Manager) Kill(id string) error {
	m.mu.Lock()
	s := m.resolve(id)
	ok := s != nil
	var infoCopy Info
	if ok {
		select {
		case <-s.done:
		default:
			if s.KillCause == "" {
				s.KillCause = "killed by user"
			}
			s.LastActivityAt = time.Now()
			s.ActivitySeq++
			infoCopy = s.Info
		}
	}
	var closeInput func(error)
	if ok {
		closeInput = s.closeInput
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("no such session: %s", id)
	}
	if infoCopy.ID != "" {
		m.emit(Event{Action: EventSessionUpdated, Info: infoCopy})
	}
	select {
	case <-s.done:
		return nil
	default:
	}
	if s.cancel != nil {
		s.cancel()
		if closeInput != nil {
			closeInput(context.Canceled)
		}
		return nil
	}
	go m.forceKill(s, s.done)
	return nil
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	running := make([]*session, 0, len(m.sessions))
	for _, s := range m.sessions {
		select {
		case <-s.done:
		default:
			running = append(running, s)
		}
	}
	m.mu.Unlock()

	for _, s := range running {
		m.setKillCause(s, "shutdown")
		if s.cancel != nil {
			s.cancel()
			if s.closeInput != nil {
				s.closeInput(context.Canceled)
			}
		} else if s.pty != nil {
			_ = s.pty.Signal(false)
		}
	}
	deadline := time.After(killGrace)
	for _, s := range running {
		select {
		case <-s.done:
		case <-deadline:
			if s.pty != nil {
				_ = s.pty.Signal(true)
			}
		}
	}
	finalDeadline := time.After(shutdownGrace)
	for _, s := range running {
		select {
		case <-s.done:
		case <-finalDeadline:
		}
	}
}

// --- query methods ---

func (m *Manager) List() []Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Info, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.Info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
}

func (m *Manager) Get(id string) (Info, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.resolve(id)
	if s == nil {
		return Info{}, false
	}
	return s.Info, true
}

func (m *Manager) SetKind(id, kind string) {
	if kind == "" {
		return
	}
	var infoCopy Info
	m.mu.Lock()
	if s := m.resolve(id); s != nil {
		s.Kind = kind
		s.LastActivityAt = time.Now()
		s.ActivitySeq++
		infoCopy = s.Info
	}
	m.mu.Unlock()
	if infoCopy.ID != "" {
		m.emit(Event{Action: EventSessionUpdated, Info: infoCopy})
	}
}

// resolve finds a session by ID first, then by name.
func (m *Manager) resolve(idOrName string) *session {
	if s, ok := m.sessions[idOrName]; ok {
		return s
	}
	for _, s := range m.sessions {
		if s.Name == idOrName {
			return s
		}
	}
	return nil
}

func (m *Manager) RunningCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, s := range m.sessions {
		if s.State == StateRunning {
			n++
		}
	}
	return n
}

func (m *Manager) Done(id string) <-chan struct{} {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return s.done
}

func (m *Manager) Peek(id string, n int) (string, error) {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return "", fmt.Errorf("no such session: %s", id)
	}
	if n <= 0 {
		n = 30
	}
	return s.output.TailLines(n), nil
}

func (m *Manager) PeekBytes(id string, n int) (string, error) {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return "", fmt.Errorf("no such session: %s", id)
	}
	return s.output.TailBytes(n), nil
}

// SnapshotBytes returns the last n bytes and the current output offset.
func (m *Manager) SnapshotBytes(id string, n int) ([]byte, int64, error) {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return nil, 0, fmt.Errorf("no such session: %s", id)
	}
	data, offset := s.output.TailRawBytesWithOffset(n)
	return data, offset, nil
}

func (m *Manager) OutputLen(id string) (int64, error) {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return 0, fmt.Errorf("no such session: %s", id)
	}
	return s.output.Len(), nil
}

func (m *Manager) PeekOrEmpty(id string, n int) string {
	s, _ := m.Peek(id, n)
	return s
}

const defaultPeekNewMax int64 = 40 * 1024

func (m *Manager) PeekNew(id string, maxBytes int64) (string, bool, error) {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return "", false, fmt.Errorf("no such session: %s", id)
	}
	if maxBytes <= 0 {
		maxBytes = defaultPeekNewMax
	}

	m.mu.Lock()
	offset := s.peekOff
	m.mu.Unlock()

	data, newOff, more, err := s.output.ReadSinceLimit(offset, maxBytes)
	if err != nil {
		return "", false, err
	}

	m.mu.Lock()
	s.peekOff = newOff
	m.mu.Unlock()

	return string(data), more, nil
}

// ReadFrom reads output since the given offset without modifying session state.
// The caller tracks the returned offset for subsequent calls.
func (m *Manager) ReadFrom(id string, offset int64, maxBytes int64) (string, int64, error) {
	data, newOff, err := m.ReadBytesFrom(id, offset, maxBytes)
	if err != nil {
		return "", offset, err
	}
	return string(data), newOff, nil
}

// ReadBytesFrom reads output bytes since the given offset without modifying
// session state. The caller tracks the returned offset for subsequent calls.
func (m *Manager) ReadBytesFrom(id string, offset int64, maxBytes int64) ([]byte, int64, error) {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return nil, 0, fmt.Errorf("no such session: %s", id)
	}
	if maxBytes <= 0 {
		maxBytes = defaultPeekNewMax
	}
	data, newOff, _, err := s.output.ReadSinceLimit(offset, maxBytes)
	if err != nil {
		return nil, offset, err
	}
	return data, newOff, nil
}

// Monitor starts a goroutine that periodically reads incremental output
// and calls push with new content. Stops automatically when the session ends.
func (m *Manager) Monitor(id string, interval time.Duration, push func(output string)) {
	_ = m.MonitorFrom(context.Background(), id, 0, interval, func(output []byte) {
		push(string(output))
	})
}

// MonitorFrom starts a cancelable incremental output monitor at offset.
func (m *Manager) MonitorFrom(ctx context.Context, id string, offset int64, interval time.Duration, push func(output []byte)) error {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return fmt.Errorf("no such session: %s", id)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.done:
				if data, _, _ := m.ReadBytesFrom(id, offset, 0); len(data) > 0 {
					push(data)
				}
				return
			case <-ticker.C:
				data, newOff, err := m.ReadBytesFrom(id, offset, 0)
				if err != nil {
					return
				}
				offset = newOff
				if len(data) > 0 {
					push(data)
				}
			}
		}
	}()
	return nil
}

func (m *Manager) Write(id string, data []byte) error {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return fmt.Errorf("no such session: %s", id)
	}
	select {
	case <-s.done:
		return fmt.Errorf("session %s already finished", id)
	default:
	}
	if s.pty == nil {
		if s.input == nil {
			return fmt.Errorf("session %s does not accept input", id)
		}
		_, err := s.input.Write(data)
		return err
	}
	_, err := s.pty.Write(data)
	return err
}

func (m *Manager) Resize(id string, cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return fmt.Errorf("no such session: %s", id)
	}
	select {
	case <-s.done:
		return nil
	default:
	}
	if s.pty == nil {
		if s.input != nil {
			infoCopy, _ := m.touchSession(id, 0)
			m.emit(Event{Action: EventSessionUpdated, Info: infoCopy})
			return nil
		}
		return fmt.Errorf("session %s does not have a pty", id)
	}
	err := s.pty.Resize(cols, rows)
	if err == nil {
		infoCopy, _ := m.touchSession(id, 0)
		m.emit(Event{Action: EventSessionUpdated, Info: infoCopy})
	}
	return err
}

func (m *Manager) Wait(ctx context.Context, id string, timeout time.Duration) (Info, error) {
	m.mu.Lock()
	s := m.resolve(id)
	m.mu.Unlock()
	if s == nil {
		return Info{}, fmt.Errorf("no such session: %s", id)
	}
	var timerC <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timerC = timer.C
	}
	var ctxErr error
	select {
	case <-s.done:
	case <-timerC:
	case <-ctx.Done():
		ctxErr = ctx.Err()
	}
	m.mu.Lock()
	info := s.Info
	m.mu.Unlock()
	return info, ctxErr
}

// --- helpers ---

func (m *Manager) setKillCause(s *session, cause string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.KillCause == "" {
		s.KillCause = cause
	}
}

func (m *Manager) getKillCause(s *session) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return s.KillCause
}

func (m *Manager) watchOutput(s *session) {
	if s == nil || s.output == nil {
		return
	}
	id := s.ID
	s.output.onWrite = func(p []byte) {
		if len(p) == 0 {
			return
		}
		if info, ok := m.touchSession(id, len(p)); ok {
			m.emit(Event{Action: EventSessionOutput, Info: info, OutputBytes: len(p)})
		}
	}
}

func (m *Manager) touchSession(id string, outputBytes int) (Info, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.resolve(id)
	if s == nil {
		return Info{}, false
	}
	s.LastActivityAt = time.Now()
	s.ActivitySeq++
	s.OutputBytes += int64(outputBytes)
	return s.Info, true
}

func (m *Manager) emit(ev Event) {
	if ev.Action == "" || ev.Info.ID == "" {
		return
	}
	m.mu.Lock()
	onEvent := m.onEvent
	m.mu.Unlock()
	if onEvent != nil {
		onEvent(ev)
	}
}

func (m *Manager) bufferCap() int {
	if m.bufCap > 0 {
		return m.bufCap
	}
	return DefaultBufferCap
}

func (m *Manager) newBuffer(outputFile string, stripANSI bool) (*OutputBuffer, error) {
	cap := m.bufferCap()
	buf := &OutputBuffer{
		buf:       make([]byte, 0, min(cap, 64*1024)),
		cap:       cap,
		stripANSI: stripANSI,
	}
	if outputFile != "" {
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, fmt.Errorf("open output file: %w", err)
		}
		buf.file = f
	}
	return buf, nil
}

func genID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func labelFromCommand(cmdLine string) string {
	cmdLine = strings.TrimSpace(cmdLine)
	if i := strings.IndexAny(cmdLine, " \t\n"); i > 0 {
		cmdLine = cmdLine[:i]
	}
	if i := strings.LastIndex(cmdLine, "/"); i >= 0 {
		cmdLine = cmdLine[i+1:]
	}
	if cmdLine == "" {
		return "shell"
	}
	return cmdLine
}

func FormatCompletion(info Info, lastOutput string) string {
	duration := info.EndedAt.Sub(info.StartedAt).Round(time.Second)
	status := "completed"
	switch {
	case info.State == StateKilled:
		status = "killed"
		if info.KillCause != "" {
			status += " (" + info.KillCause + ")"
		}
	case info.ExitCode != 0:
		status = fmt.Sprintf("exited with code %d", info.ExitCode)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "<session_completion id=%q name=%q exit_code=%d duration=%q>\n",
		info.ID, info.Name, info.ExitCode, duration.String())
	fmt.Fprintf(&sb, "Background session %s.\n", status)
	if lastOutput != "" {
		sb.WriteString("--- last 20 lines ---\n")
		sb.WriteString(lastOutput)
		sb.WriteString("\n")
	} else {
		sb.WriteString("(no output)\n")
	}
	sb.WriteString("</session_completion>")
	return sb.String()
}
