package music

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"

	dsound "github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/sound"
)

type playOpener func(context.Context, dsound.Config) (io.WriteCloser, error)

type playerCmd struct {
	path  string
	frame int64
}

// player pulls a file in 20ms periods and records the frame it has written.
type player struct {
	open     playOpener
	commands chan playerCmd

	played  atomic.Int64
	total   atomic.Int64
	playing atomic.Bool

	mu   sync.Mutex
	path string
	err  string
}

func newPlayer(open playOpener) *player {
	if open == nil {
		open = dsound.Open
	}
	return &player{open: open, commands: make(chan playerCmd, 4)}
}

// Play starts path at frame. An empty path stops.
func (p *player) Play(path string, frame int64) {
	if p == nil {
		return
	}
	cmd := playerCmd{path: path, frame: frame}
	select {
	case p.commands <- cmd:
	default:
		select {
		case <-p.commands:
		default:
		}
		select {
		case p.commands <- cmd:
		default:
		}
	}
}

// Stop ends the current file. Played stays where the last period finished.
func (p *player) Stop() { p.Play("", 0) }

func (p *player) Played() int64 { return p.played.Load() }

func (p *player) Total() int64 { return p.total.Load() }

func (p *player) Playing() bool { return p.playing.Load() }

func (p *player) Path() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.path
}

func (p *player) Err() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

func (p *player) setStatus(path, errText string) {
	p.mu.Lock()
	p.path = path
	p.err = errText
	p.mu.Unlock()
}

func (p *player) loop(ctx context.Context) {
	var session *playSession
	for {
		if session == nil {
			select {
			case <-ctx.Done():
				return
			case cmd := <-p.commands:
				session = p.begin(ctx, cmd)
			}
			continue
		}
		select {
		case <-ctx.Done():
			session.close()
			return
		case cmd := <-p.commands:
			session.close()
			session = p.begin(ctx, cmd)
		default:
			if err := session.step(); err != nil {
				session.close()
				session = nil
				p.playing.Store(false)
			}
		}
	}
}

type playSession struct {
	player *player
	file   *os.File
	out    io.WriteCloser
	pipe   *sound.Pipeline
	buf    []byte
	frame  int
	start  int64
	got    int64
}

func (p *player) begin(ctx context.Context, cmd playerCmd) *playSession {
	p.playing.Store(false)
	if cmd.path == "" {
		p.setStatus("", "")
		return nil
	}
	file, err := os.Open(cmd.path)
	if err != nil {
		p.setStatus(cmd.path, err.Error())
		return nil
	}
	pipe, err := sound.Decode(cmd.path, file)
	if err != nil {
		p.fail(cmd.path, errors.Join(err, file.Close()))
		return nil
	}
	if cmd.frame > 0 {
		if _, err := pipe.Seek(cmd.frame, io.SeekStart); err != nil {
			p.fail(cmd.path, errors.Join(err, file.Close()))
			return nil
		}
	}
	out, err := p.open(ctx, dsound.Config{Format: pipe.Format()})
	if err != nil {
		p.fail(cmd.path, errors.Join(err, file.Close()))
		return nil
	}
	frame, err := pipe.Format().Frame()
	if err != nil {
		p.fail(cmd.path, errors.Join(err, out.Close(), file.Close()))
		return nil
	}
	period := pipe.Format().Rate * frame / 50
	if period < frame {
		period = frame
	}
	period -= period % frame
	p.total.Store(pipe.Frames())
	p.played.Store(cmd.frame)
	p.playing.Store(true)
	p.setStatus(cmd.path, "")
	return &playSession{
		player: p,
		file:   file,
		out:    out,
		pipe:   pipe,
		buf:    make([]byte, period),
		frame:  frame,
		start:  cmd.frame,
	}
}

func (session *playSession) step() error {
	n, err := session.pipe.Read(session.buf)
	if n > 0 {
		if _, writeErr := session.out.Write(session.buf[:n]); writeErr != nil {
			return writeErr
		}
		session.got += int64(n / session.frame)
		session.player.played.Store(session.start + session.got)
	}
	if err == io.EOF {
		return io.EOF
	}
	return err
}

func (p *player) fail(path string, err error) {
	if err == nil {
		p.setStatus(path, "")
		return
	}
	p.setStatus(path, err.Error())
}

func (session *playSession) close() {
	if session == nil {
		return
	}
	var err error
	if session.out != nil {
		err = session.out.Close()
	}
	if session.file != nil {
		err = errors.Join(err, session.file.Close())
	}
	if err != nil && session.player.Err() == "" {
		session.player.setStatus(session.player.Path(), err.Error())
	}
}
