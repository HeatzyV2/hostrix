package console

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hostrix/hostrix/agent/internal/containers"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

// AttachInteractiveShell streams an interactive shell between the WebSocket and Incus.
func AttachInteractiveShell(ctx context.Context, srv incus.InstanceServer, name string, cols, rows int, client *websocket.Conn) error {
	if err := containers.ValidateContainerName(name); err != nil {
		return err
	}
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	var (
		controlMu sync.Mutex
		control   *websocket.Conn
	)

	dataDone := make(chan bool)
	args := &incus.InstanceExecArgs{
		Stdin:    stdinReader,
		Stdout:   stdoutWriter,
		Stderr:   stdoutWriter,
		DataDone: dataDone,
		Control: func(conn *websocket.Conn) {
			controlMu.Lock()
			control = conn
			controlMu.Unlock()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		},
	}

	op, err := srv.ExecInstance(name, api.InstanceExecPost{
		Command:     []string{"/bin/bash", "-l"},
		WaitForWS:   true,
		Interactive: true,
		Width:       cols,
		Height:      rows,
	}, args)
	if err != nil {
		_ = stdinWriter.Close()
		_ = stdoutWriter.Close()
		return fmt.Errorf("exec: %w", err)
	}

	errCh := make(chan error, 3)

	go func() {
		defer stdinWriter.Close()
		for {
			msgType, data, err := client.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			if msgType == websocket.TextMessage && len(data) > 0 && data[0] == '{' {
				var ctrl struct {
					Type string `json:"type"`
					Cols int    `json:"cols"`
					Rows int    `json:"rows"`
				}
				if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" {
					sendResize(&controlMu, &control, ctrl.Cols, ctrl.Rows)
					continue
				}
			}
			if _, err := stdinWriter.Write(data); err != nil {
				errCh <- err
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutReader.Read(buf)
			if n > 0 {
				if werr := client.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					errCh <- werr
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					errCh <- err
				} else {
					errCh <- nil
				}
				return
			}
		}
	}()

	go func() {
		errCh <- op.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = stdinWriter.Close()
		_ = stdoutWriter.Close()
		_ = client.Close()
		return ctx.Err()
	case err := <-errCh:
		_ = stdinWriter.Close()
		_ = stdoutWriter.Close()
		select {
		case <-dataDone:
		default:
		}
		return err
	}
}

func sendResize(mu *sync.Mutex, control **websocket.Conn, cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	mu.Lock()
	conn := *control
	mu.Unlock()
	if conn == nil {
		return
	}
	_ = conn.WriteJSON(map[string]any{
		"command": "window-resize",
		"args": map[string]string{
			"width":  fmt.Sprintf("%d", cols),
			"height": fmt.Sprintf("%d", rows),
		},
	})
}
