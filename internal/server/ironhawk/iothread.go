// Copyright (c) 2022-present, DiceDB contributors
// All rights reserved. Licensed under the BSD 3-Clause License.

package ironhawk

import (
	"context"
	"log/slog"
	"strings"

	"github.com/dicedb/dicedb-go"
	"github.com/dicedb/dicedb-go/wire"

	"github.com/dicedb/dice/config"
	"github.com/dicedb/dice/internal/auth"
	"github.com/dicedb/dice/internal/cmd"
	"github.com/dicedb/dice/internal/shardmanager"
	"github.com/dicedb/dice/internal/wal"
)

const maxWireSize = 32 * 1024 * 1024 // 32MB safety limit

type IOThread struct {
	ClientID   string
	Mode       string
	Session    *auth.Session
	serverWire *dicedb.ServerWire
}

func NewIOThread(clientFD int) (*IOThread, error) {
	w, err := dicedb.NewServerWire(config.MaxRequestSize, config.KeepAlive, clientFD)
	if err != nil {
		if err.Kind == wire.NotEstablished {
			slog.Error("failed to establish connection to client", slog.String("error", err.Error()))
			return nil, err.Unwrap()
		}
		slog.Error("unexpected error during client connection establishment", slog.String("error", err.Error()))
		return nil, err.Unwrap()
	}

	return &IOThread{
		serverWire: w,
		Session:    auth.NewSession(),
	}, nil
}

func (t *IOThread) safeSend(ctx context.Context, res *wire.Result) error {
	if res == nil {
		return nil
	}

	// hard limit protection
	if len(res.Message) > maxWireSize {
		slog.Error("dropping oversized response",
			slog.Int("size", len(res.Message)),
		)

		res = &wire.Result{
			Status:  wire.Status_ERR,
			Message: "response too large (truncated by server)",
		}
	}

	return t.serverWire.Send(ctx, res)
}

func (t *IOThread) Start(ctx context.Context, shardManager *shardmanager.ShardManager, watchManager WatchManager) error {
	for {
		var c *wire.Command
		recvCh := make(chan *wire.Command, 1)
		errCh := make(chan error, 1)

		go func() {
			tmpC, err := t.serverWire.Receive()
			if err != nil {
				errCh <- err.Unwrap()
				return
			}
			recvCh <- tmpC
		}()

		select {
		case <-ctx.Done():
			slog.Debug("io-thread context canceled, shutting down")
			return ctx.Err()

		case err := <-errCh:
			return err

		case tmp := <-recvCh:
			c = tmp
		}

		_c := &cmd.Cmd{
			C:        c,
			ClientID: t.ClientID,
			Mode:     t.Mode,
		}

		res, err := _c.Execute(shardManager)

		// ---------------- ERROR HANDLING ----------------
		if err != nil {
			res = &cmd.CmdRes{
				Rs: &wire.Result{
					Status:  wire.Status_ERR,
					Message: err.Error(),
				},
			}

			if sendErr := t.safeSend(ctx, res.Rs); sendErr != nil {
				return sendErr
			}
			continue
		}

		// ---------------- NORMAL RESPONSE ----------------
		res.Rs.Status = wire.Status_OK
		if res.Rs.Message == "" {
			res.Rs.Message = "OK"
		}

		// ---------------- WAL ----------------
		if wal.DefaultWAL != nil && !_c.IsReplay {
			if err := wal.DefaultWAL.LogCommand(_c.C); err != nil {
				slog.Error("failed to log command to WAL", slog.String("error", err.Error()))
			}
		}

		if err == nil {
			t.ClientID = _c.ClientID
		}

		if _c.Meta.IsWatchable {
			tmp := _c
			tmp.C.Cmd += ".WATCH"
			res.Rs.Fingerprint64 = tmp.Fingerprint()
		}

		// ---------------- HANDSHAKE ----------------
		if c.Cmd == "HANDSHAKE" && err == nil {
			t.ClientID = _c.C.Args[0]
			t.Mode = _c.C.Args[1]
		}

		isWatchCmd := strings.HasSuffix(c.Cmd, "WATCH")

		// ---------------- WATCH HANDLING ----------------
		if isWatchCmd {
			go func(cmd *cmd.Cmd) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("watch panic recovered", slog.Any("err", r))
					}
				}()
				watchManager.HandleWatch(cmd, t)
			}(_c)

		} else if strings.HasSuffix(c.Cmd, "UNWATCH") {
			watchManager.HandleUnwatch(_c, t)
		}

		watchManager.RegisterThread(t)

		// ---------------- SEND RESPONSE ----------------
		if !isWatchCmd {
			if sendErr := t.safeSend(ctx, res.Rs); sendErr != nil {
				return sendErr
			}
		}

		// ---------------- WATCH NOTIFY ----------------
		if err == nil {
			go func(cmd *cmd.Cmd) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("notify panic", slog.Any("err", r))
					}
				}()
				watchManager.NotifyWatchers(cmd, shardManager, t)
			}(_c)
		}
	}
}

func (t *IOThread) Stop() error {
	t.serverWire.Close()
	t.Session.Expire()
	return nil
}
