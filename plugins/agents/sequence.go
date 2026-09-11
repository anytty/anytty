package agents

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// StampCodex assigns sequence in adapter-observation order, not undocumented
// Agent-internal event order. Official Codex hook stdin contains no source
// timestamp/sequence; concurrent callbacks cannot reconstruct that ordering.
func StampCodex(ctx context.Context, stateDir string, e *Event) error {
	if e == nil {
		return nil
	}
	if e.Agent != "codex" || e.SessionID == "" || stateDir == "" {
		return errors.New("invalid Codex sequencing context")
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(e.Agent+"\x00"+e.SessionID)))
	lock, err := os.OpenFile(filepath.Join(stateDir, key+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	for {
		err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	path := filepath.Join(stateDir, key+".json")
	var clock struct {
		Epoch    uint64
		Sequence uint64
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		if err = json.Unmarshal(data, &clock); err != nil {
			return err
		}
	}
	if clock.Epoch == 0 || e.Kind == "start" {
		next := uint64(time.Now().UnixMilli())
		if next <= clock.Epoch {
			next = clock.Epoch + 1
		}
		clock.Epoch = next
		clock.Sequence = 0
	}
	clock.Sequence++
	e.Epoch = clock.Epoch
	e.Sequence = clock.Sequence
	data, err = json.Marshal(clock)
	if err != nil {
		return err
	}
	return atomicWrite(path, data, 0600)
}
