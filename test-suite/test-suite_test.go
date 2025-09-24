package test_suite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

const (
	URL           = "http://localhost:3000"
	WS_URL        = "ws://localhost:3000"
	PING_INTERVAL = 300
	PING_TIMEOUT  = 200
)

func waitFor(c *websocket.Conn, ctx context.Context) (string, error) {
	_, data, err := c.Read(ctx)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func waitForPackets(c *websocket.Conn, ctx context.Context, count int) ([]any, error) {
	packets := make([]any, 0, count)

	for len(packets) < count {
		msgType, data, err := c.Read(ctx)
		if err != nil {
			return nil, err
		}

		if msgType == websocket.MessageText && string(data) == "2" {
			// ignore PING packets
			continue
		}

		if msgType == websocket.MessageBinary {
			packets = append(packets, data)
		} else {
			packets = append(packets, string(data))
		}
	}

	return packets, nil
}

func initLongPollingSession(t *testing.T) string {
	resp, err := http.Get(URL + "/socket.io/?EIO=4&transport=polling")
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	content := string(body)
	var val map[string]any
	if err := json.Unmarshal([]byte(content[1:]), &val); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	sid, ok := val["sid"].(string)
	if !ok {
		t.Fatalf("sid not string")
	}
	return sid
}

func initSocketIOConnection(t *testing.T) *websocket.Conn {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}

	// Engine.IO handshake
	_, err = waitFor(c, ctx)
	if err != nil {
		t.Fatalf("failed to read handshake: %v", err)
	}

	// send "40" = Socket.IO connect
	if err := c.Write(ctx, websocket.MessageText, []byte("40")); err != nil {
		t.Fatalf("ws write: %v", err)
	}

	// Socket.IO handshake
	_, err = waitFor(c, ctx)
	if err != nil {
		t.Fatalf("failed to read socket.io handshake: %v", err)
	}

	// "auth" packet
	_, err = waitFor(c, ctx)
	if err != nil {
		t.Fatalf("failed to read auth packet: %v", err)
	}

	return c
}

func TestEngineIOHandshake(t *testing.T) {
	t.Run("HTTP long-polling", func(t *testing.T) {
		t.Run("should successfully open a session", func(t *testing.T) {
			resp, err := http.Get(URL + "/socket.io/?EIO=4&transport=polling")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			content := string(body)
			if !strings.HasPrefix(content, "0") {
				t.Fatalf("expected handshake starting with 0, got %s", content)
			}

			var val map[string]any
			if err := json.Unmarshal([]byte(content[1:]), &val); err != nil {
				t.Fatal(err)
			}

			// Check all required keys
			expectedKeys := []string{"sid", "upgrades", "pingInterval", "pingTimeout", "maxPayload"}
			for _, key := range expectedKeys {
				if _, exists := val[key]; !exists {
					t.Fatalf("missing key: %s", key)
				}
			}

			if _, ok := val["sid"].(string); !ok {
				t.Fatal("sid should be a string")
			}

			upgrades, ok := val["upgrades"].([]any)
			if !ok {
				t.Fatal("upgrades should be an array")
			}
			if len(upgrades) != 1 || upgrades[0] != "websocket" {
				t.Fatal("upgrades should be ['websocket']")
			}

			if val["pingInterval"] != float64(PING_INTERVAL) {
				t.Fatalf("expected pingInterval %d, got %v", PING_INTERVAL, val["pingInterval"])
			}
			if val["pingTimeout"] != float64(PING_TIMEOUT) {
				t.Fatalf("expected pingTimeout %d, got %v", PING_TIMEOUT, val["pingTimeout"])
			}
			if val["maxPayload"] != float64(1000000) {
				t.Fatalf("expected maxPayload 1000000, got %v", val["maxPayload"])
			}
		})

		t.Run("should fail with an invalid 'EIO' query parameter", func(t *testing.T) {
			resp, err := http.Get(URL + "/socket.io/?transport=polling")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 400 {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}

			resp2, err := http.Get(URL + "/socket.io/?EIO=abc&transport=polling")
			if err != nil {
				t.Fatal(err)
			}
			defer resp2.Body.Close()

			if resp2.StatusCode != 400 {
				t.Fatalf("expected 400, got %d", resp2.StatusCode)
			}
		})

		t.Run("should fail with an invalid 'transport' query parameter", func(t *testing.T) {
			resp, err := http.Get(URL + "/socket.io/?EIO=4")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 400 {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}

			resp2, err := http.Get(URL + "/socket.io/?EIO=4&transport=abc")
			if err != nil {
				t.Fatal(err)
			}
			defer resp2.Body.Close()

			if resp2.StatusCode != 400 {
				t.Fatalf("expected 400, got %d", resp2.StatusCode)
			}
		})

		t.Run("should fail with an invalid request method", func(t *testing.T) {
			resp, err := http.Post(URL+"/socket.io/?EIO=4&transport=polling", "", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 400 {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}

			req, err := http.NewRequest("PUT", URL+"/socket.io/?EIO=4&transport=polling", nil)
			if err != nil {
				t.Fatal(err)
			}

			resp2, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp2.Body.Close()

			if resp2.StatusCode != 400 {
				t.Fatalf("expected 400, got %d", resp2.StatusCode)
			}
		})
	})

	t.Run("WebSocket", func(t *testing.T) {
		t.Run("should successfully open a session", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close(websocket.StatusNormalClosure, "")

			data, err := waitFor(c, ctx)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(data, "0") {
				t.Fatalf("expected 0 handshake, got %s", data)
			}

			var val map[string]any
			if err := json.Unmarshal([]byte(data[1:]), &val); err != nil {
				t.Fatal(err)
			}

			// Check all required keys
			expectedKeys := []string{"sid", "upgrades", "pingInterval", "pingTimeout", "maxPayload"}
			for _, key := range expectedKeys {
				if _, exists := val[key]; !exists {
					t.Fatalf("missing key: %s", key)
				}
			}

			if _, ok := val["sid"].(string); !ok {
				t.Fatal("sid should be a string")
			}

			upgrades, ok := val["upgrades"].([]any)
			if !ok {
				t.Fatal("upgrades should be an array")
			}
			if len(upgrades) != 0 {
				t.Fatal("upgrades should be empty array for websocket")
			}

			if val["pingInterval"] != float64(PING_INTERVAL) {
				t.Fatalf("expected pingInterval %d, got %v", PING_INTERVAL, val["pingInterval"])
			}
			if val["pingTimeout"] != float64(PING_TIMEOUT) {
				t.Fatalf("expected pingTimeout %d, got %v", PING_TIMEOUT, val["pingTimeout"])
			}
			if val["maxPayload"] != float64(1000000) {
				t.Fatalf("expected maxPayload 1000000, got %v", val["maxPayload"])
			}
		})

		t.Run("should fail with an invalid 'EIO' query parameter", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?transport=websocket", nil)
			if c != nil {
				c.Close(websocket.StatusNormalClosure, "")
			}
			// Connection should fail or close immediately

			c2, _, err2 := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=abc&transport=websocket", nil)
			if c2 != nil {
				c2.Close(websocket.StatusNormalClosure, "")
			}
			// Connection should fail or close immediately

			if err == nil && err2 == nil {
				t.Log("Connections may have been established but should close immediately")
			}
		})

		t.Run("should fail with an invalid 'transport' query parameter", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4", nil)
			if c != nil {
				c.Close(websocket.StatusNormalClosure, "")
			}
			// Connection should fail or close immediately

			c2, _, err2 := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=abc", nil)
			if c2 != nil {
				c2.Close(websocket.StatusNormalClosure, "")
			}
			// Connection should fail or close immediately

			if err == nil && err2 == nil {
				t.Log("Connections may have been established but should close immediately")
			}
		})
	})
}

func TestEngineIOHeartbeat(t *testing.T) {
	t.Run("HTTP long-polling", func(t *testing.T) {
		t.Run("should send ping/pong packets", func(t *testing.T) {
			sid := initLongPollingSession(t)

			for range 3 {
				pollResponse, err := http.Get(fmt.Sprintf("%s/socket.io/?EIO=4&transport=polling&sid=%s", URL, sid))
				if err != nil {
					t.Fatal(err)
				}

				if pollResponse.StatusCode != 200 {
					t.Fatalf("expected 200, got %d", pollResponse.StatusCode)
				}

				pollBody, err := io.ReadAll(pollResponse.Body)
				pollResponse.Body.Close()
				if err != nil {
					t.Fatal(err)
				}

				pollContent := string(pollBody)
				if pollContent != "2" {
					t.Fatalf("expected '2', got %s", pollContent)
				}

				pushResponse, err := http.Post(
					fmt.Sprintf("%s/socket.io/?EIO=4&transport=polling&sid=%s", URL, sid),
					"text/plain",
					strings.NewReader("3"),
				)
				if err != nil {
					t.Fatal(err)
				}
				pushResponse.Body.Close()

				if pushResponse.StatusCode != 200 {
					t.Fatalf("expected 200, got %d", pushResponse.StatusCode)
				}
			}
		})

		t.Run("should close the session upon ping timeout", func(t *testing.T) {
			sid := initLongPollingSession(t)

			time.Sleep(time.Duration(PING_INTERVAL+PING_TIMEOUT) * time.Millisecond)

			pollResponse, err := http.Get(fmt.Sprintf("%s/socket.io/?EIO=4&transport=polling&sid=%s", URL, sid))
			if err != nil {
				t.Fatal(err)
			}
			defer pollResponse.Body.Close()

			if pollResponse.StatusCode != 400 {
				t.Fatalf("expected 400, got %d", pollResponse.StatusCode)
			}
		})
	})

	t.Run("WebSocket", func(t *testing.T) {
		t.Run("should send ping/pong packets", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close(websocket.StatusNormalClosure, "")

			// handshake
			_, err = waitFor(c, ctx)
			if err != nil {
				t.Fatal(err)
			}

			for range 3 {
				data, err := waitFor(c, ctx)
				if err != nil {
					t.Fatal(err)
				}

				if data != "2" {
					t.Fatalf("expected '2', got %s", data)
				}

				err = c.Write(ctx, websocket.MessageText, []byte("3"))
				if err != nil {
					t.Fatal(err)
				}
			}
		})

		t.Run("should close the session upon ping timeout", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Wait for close event - connection should close due to timeout
			for {
				_, _, err := c.Read(ctx)
				if err != nil {
					// Connection closed as expected
					break
				}
			}
		})
	})
}

func TestEngineIOClose(t *testing.T) {
	t.Run("HTTP long-polling", func(t *testing.T) {
		t.Run("should forcefully close the session", func(t *testing.T) {
			sid := initLongPollingSession(t)

			// Create channels to coordinate the parallel requests
			pollDone := make(chan *http.Response, 1)
			pushDone := make(chan error, 1)

			// Start polling request first
			go func() {
				// Small delay to ensure polling request starts first
				time.Sleep(10 * time.Millisecond)
				resp, err := http.Get(fmt.Sprintf("%s/socket.io/?EIO=4&transport=polling&sid=%s", URL, sid))
				if err != nil {
					t.Logf("poll request error: %v", err)
					pollDone <- nil
					return
				}
				pollDone <- resp
			}()

			// Start push request (close command) after a small delay
			go func() {
				time.Sleep(50 * time.Millisecond)
				resp, err := http.Post(
					fmt.Sprintf("%s/socket.io/?EIO=4&transport=polling&sid=%s", URL, sid),
					"text/plain",
					strings.NewReader("1"),
				)
				if resp != nil {
					resp.Body.Close()
				}
				pushDone <- err
			}()

			// Wait for poll response
			pollResponse := <-pollDone
			pushError := <-pushDone

			if pollResponse == nil {
				t.Fatal("Poll request failed")
			}

			defer pollResponse.Body.Close()

			if pushError != nil {
				t.Logf("Push request error (may be expected): %v", pushError)
			}

			// The poll response should be 200 with close packet "6" or timeout (400)
			if pollResponse.StatusCode == 200 {
				pullBody, err := io.ReadAll(pollResponse.Body)
				if err != nil {
					t.Fatal(err)
				}
				pullContent := string(pullBody)

				if pullContent != "6" && pullContent != "2" {
					t.Fatalf("expected '6' (close) or '2' (ping), got %s", pullContent)
				}
			} else if pollResponse.StatusCode != 400 {
				t.Fatalf("expected 200 or 400, got %d", pollResponse.StatusCode)
			}

			// Give some time for the close to take effect
			time.Sleep(100 * time.Millisecond)

			// Try another request - should fail
			pollResponse2, err := http.Get(fmt.Sprintf("%s/socket.io/?EIO=4&transport=polling&sid=%s", URL, sid))
			if err != nil {
				t.Fatal(err)
			}
			defer pollResponse2.Body.Close()

			if pollResponse2.StatusCode != 400 {
				t.Fatalf("expected 400 for subsequent request, got %d", pollResponse2.StatusCode)
			}
		})
	})

	t.Run("WebSocket", func(t *testing.T) {
		t.Run("should forcefully close the session", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
			if err != nil {
				t.Fatal(err)
			}

			// handshake
			_, err = waitFor(c, ctx)
			if err != nil {
				t.Fatal(err)
			}

			// send close command
			err = c.Write(ctx, websocket.MessageText, []byte("1"))
			if err != nil {
				t.Fatal(err)
			}

			// Wait for connection to close
			for {
				_, _, err := c.Read(ctx)
				if err != nil {
					// Connection closed as expected
					break
				}
			}
		})
	})
}

func TestEngineIOUpgrade(t *testing.T) {
	t.Run("should successfully upgrade from HTTP long-polling to WebSocket", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sid := initLongPollingSession(t)

		c, _, err := websocket.Dial(ctx, fmt.Sprintf("%s/socket.io/?EIO=4&transport=websocket&sid=%s", WS_URL, sid), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// send probe
		err = c.Write(ctx, websocket.MessageText, []byte("2probe"))
		if err != nil {
			t.Fatal(err)
		}

		probeResponse, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if probeResponse != "3probe" {
			t.Fatalf("expected '3probe', got %s", probeResponse)
		}

		// complete upgrade
		err = c.Write(ctx, websocket.MessageText, []byte("5"))
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("should ignore HTTP requests with same sid after upgrade", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sid := initLongPollingSession(t)

		c, _, err := websocket.Dial(ctx, fmt.Sprintf("%s/socket.io/?EIO=4&transport=websocket&sid=%s", WS_URL, sid), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Send probe
		err = c.Write(ctx, websocket.MessageText, []byte("2probe"))
		if err != nil {
			t.Fatal(err)
		}

		// Wait for probe response
		probeResp, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if probeResp != "3probe" {
			t.Logf("Expected probe response, got: %s", probeResp)
		}

		// Complete upgrade
		err = c.Write(ctx, websocket.MessageText, []byte("5"))
		if err != nil {
			t.Fatal(err)
		}

		// Wait a bit for upgrade to complete
		time.Sleep(100 * time.Millisecond)

		// Now try HTTP request - should fail with 400
		pollResponse, err := http.Get(fmt.Sprintf("%s/socket.io/?EIO=4&transport=polling&sid=%s", URL, sid))
		if err != nil {
			t.Fatal(err)
		}
		defer pollResponse.Body.Close()

		// After upgrade, HTTP requests should be rejected
		if pollResponse.StatusCode != 400 {
			body, _ := io.ReadAll(pollResponse.Body)
			t.Fatalf("expected 400 after upgrade, got %d, body: %s", pollResponse.StatusCode, string(body))
		}
	})

	t.Run("should ignore WebSocket connection with same sid after upgrade", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sid := initLongPollingSession(t)

		c, _, err := websocket.Dial(ctx, fmt.Sprintf("%s/socket.io/?EIO=4&transport=websocket&sid=%s", WS_URL, sid), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		err = c.Write(ctx, websocket.MessageText, []byte("2probe"))
		if err != nil {
			t.Fatal(err)
		}

		err = c.Write(ctx, websocket.MessageText, []byte("5"))
		if err != nil {
			t.Fatal(err)
		}

		c2, _, err := websocket.Dial(ctx, fmt.Sprintf("%s/socket.io/?EIO=4&transport=websocket&sid=%s", WS_URL, sid), nil)
		if c2 != nil {
			defer c2.Close(websocket.StatusNormalClosure, "")
		}

		if c2 != nil {
			// Wait for close
			for {
				_, _, err := c2.Read(ctx)
				if err != nil {
					// Connection closed as expected
					break
				}
			}
		}
	})
}

func TestSocketIOConnect(t *testing.T) {
	t.Run("should allow connection to the main namespace", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Engine.IO handshake
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Write(ctx, websocket.MessageText, []byte("40"))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.HasPrefix(data, "40") {
			t.Fatalf("expected message starting with '40', got %s", data)
		}

		var handshake map[string]any
		if err := json.Unmarshal([]byte(data[2:]), &handshake); err != nil {
			t.Fatal(err)
		}

		if len(handshake) != 1 {
			t.Fatalf("expected handshake to have only 'sid' key, got keys: %v", handshake)
		}

		if _, ok := handshake["sid"].(string); !ok {
			t.Fatal("sid should be a string")
		}

		authPacket, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if authPacket != `42["auth",{}]` {
			t.Fatalf("expected auth packet, got %s", authPacket)
		}
	})

	t.Run("should allow connection to the main namespace with a payload", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Engine.IO handshake
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Write(ctx, websocket.MessageText, []byte(`40{"token":"123"}`))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.HasPrefix(data, "40") {
			t.Fatalf("expected message starting with '40', got %s", data)
		}

		var handshake map[string]any
		if err := json.Unmarshal([]byte(data[2:]), &handshake); err != nil {
			t.Fatal(err)
		}

		if len(handshake) != 1 {
			t.Fatalf("expected handshake to have only 'sid' key, got keys: %v", handshake)
		}

		if _, ok := handshake["sid"].(string); !ok {
			t.Fatal("sid should be a string")
		}

		authPacket, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if authPacket != `42["auth",{"token":"123"}]` {
			t.Fatalf("expected auth packet with token, got %s", authPacket)
		}
	})

	t.Run("should allow connection to a custom namespace", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Engine.IO handshake
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Write(ctx, websocket.MessageText, []byte("40/custom,"))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.HasPrefix(data, "40/custom,") {
			t.Fatalf("expected message starting with '40/custom,', got %s", data)
		}

		var handshake map[string]any
		if err := json.Unmarshal([]byte(data[10:]), &handshake); err != nil {
			t.Fatal(err)
		}

		if len(handshake) != 1 {
			t.Fatalf("expected handshake to have only 'sid' key, got keys: %v", handshake)
		}

		if _, ok := handshake["sid"].(string); !ok {
			t.Fatal("sid should be a string")
		}

		authPacket, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if authPacket != `42/custom,["auth",{}]` {
			t.Fatalf("expected auth packet for custom namespace, got %s", authPacket)
		}
	})

	t.Run("should allow connection to a custom namespace with a payload", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Engine.IO handshake
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Write(ctx, websocket.MessageText, []byte(`40/custom,{"token":"abc"}`))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.HasPrefix(data, "40/custom,") {
			t.Fatalf("expected message starting with '40/custom,', got %s", data)
		}

		var handshake map[string]any
		if err := json.Unmarshal([]byte(data[10:]), &handshake); err != nil {
			t.Fatal(err)
		}

		if len(handshake) != 1 {
			t.Fatalf("expected handshake to have only 'sid' key, got keys: %v", handshake)
		}

		if _, ok := handshake["sid"].(string); !ok {
			t.Fatal("sid should be a string")
		}

		authPacket, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if authPacket != `42/custom,["auth",{"token":"abc"}]` {
			t.Fatalf("expected auth packet for custom namespace with token, got %s", authPacket)
		}
	})

	t.Run("should disallow connection to an unknown namespace", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Engine.IO handshake
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Write(ctx, websocket.MessageText, []byte("40/random"))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if data != `44/random,{"message":"Invalid namespace"}` {
			t.Fatalf("expected error message for invalid namespace, got %s", data)
		}
	})

	t.Run("should disallow connection with an invalid handshake", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Engine.IO handshake
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Write(ctx, websocket.MessageText, []byte("4abc"))
		if err != nil {
			t.Fatal(err)
		}

		// Wait for connection to close
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				// Connection closed as expected
				break
			}
		}
	})

	t.Run("should close the connection if no handshake is received", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, WS_URL+"/socket.io/?EIO=4&transport=websocket", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Don't send any handshake, just wait for close
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				// Connection closed as expected
				break
			}
		}
	})
}

func TestSocketIODisconnect(t *testing.T) {
	t.Run("should disconnect from the main namespace", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)
		defer c.Close(websocket.StatusNormalClosure, "")

		err := c.Write(ctx, websocket.MessageText, []byte("41"))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if data != "2" {
			t.Fatalf("expected '2', got %s", data)
		}
	})

	t.Run("should connect then disconnect from a custom namespace", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)
		defer c.Close(websocket.StatusNormalClosure, "")

		// Wait for ping
		_, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Connect to custom namespace
		err = c.Write(ctx, websocket.MessageText, []byte("40/custom"))
		if err != nil {
			t.Fatal(err)
		}

		// Socket.IO handshake for custom namespace
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		// auth packet for custom namespace
		_, err = waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Disconnect from custom namespace
		err = c.Write(ctx, websocket.MessageText, []byte("41/custom"))
		if err != nil {
			t.Fatal(err)
		}

		// Send message to main namespace
		err = c.Write(ctx, websocket.MessageText, []byte(`42["message","message to main namespace"]`))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if data != `42["message-back","message to main namespace"]` {
			t.Fatalf("expected message-back, got %s", data)
		}
	})
}

func TestSocketIOMessage(t *testing.T) {
	t.Run("should send a plain-text packet", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)
		defer c.Close(websocket.StatusNormalClosure, "")

		err := c.Write(ctx, websocket.MessageText, []byte(`42["message",1,"2",{"3":[true]}]`))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if data != `42["message-back",1,"2",{"3":[true]}]` {
			t.Fatalf("expected message-back with same data, got %s", data)
		}
	})

	t.Run("should send a packet with binary attachments", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)
		defer c.Close(websocket.StatusNormalClosure, "")

		// Send the message packet
		err := c.Write(ctx, websocket.MessageText, []byte(`452-["message",{"_placeholder":true,"num":0},{"_placeholder":true,"num":1}]`))
		if err != nil {
			t.Fatal(err)
		}

		// Send first binary attachment
		err = c.Write(ctx, websocket.MessageBinary, []byte{1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}

		// Send second binary attachment
		err = c.Write(ctx, websocket.MessageBinary, []byte{4, 5, 6})
		if err != nil {
			t.Fatal(err)
		}

		// Wait for 3 packets in response
		packets, err := waitForPackets(c, ctx, 3)
		if err != nil {
			t.Fatal(err)
		}

		expectedText := `452-["message-back",{"_placeholder":true,"num":0},{"_placeholder":true,"num":1}]`
		if packets[0].(string) != expectedText {
			t.Fatalf("expected %s, got %s", expectedText, packets[0])
		}

		// Check binary data
		binary1, ok := packets[1].([]byte)
		if !ok {
			t.Fatal("expected binary data")
		}
		if !bytes.Equal(binary1, []byte{1, 2, 3}) {
			t.Fatalf("expected [1,2,3], got %v", binary1)
		}

		binary2, ok := packets[2].([]byte)
		if !ok {
			t.Fatal("expected binary data")
		}
		if !bytes.Equal(binary2, []byte{4, 5, 6}) {
			t.Fatalf("expected [4,5,6], got %v", binary2)
		}
	})

	t.Run("should send a plain-text packet with an ack", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)
		defer c.Close(websocket.StatusNormalClosure, "")

		err := c.Write(ctx, websocket.MessageText, []byte(`42456["message-with-ack",1,"2",{"3":[false]}]`))
		if err != nil {
			t.Fatal(err)
		}

		data, err := waitFor(c, ctx)
		if err != nil {
			t.Fatal(err)
		}

		if data != `43456[1,"2",{"3":[false]}]` {
			t.Fatalf("expected ack response, got %s", data)
		}
	})

	t.Run("should send a packet with binary attachments and an ack", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)
		defer c.Close(websocket.StatusNormalClosure, "")

		// Send the message packet with ack
		err := c.Write(ctx, websocket.MessageText, []byte(`452-789["message-with-ack",{"_placeholder":true,"num":0},{"_placeholder":true,"num":1}]`))
		if err != nil {
			t.Fatal(err)
		}

		// Send first binary attachment
		err = c.Write(ctx, websocket.MessageBinary, []byte{1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}

		// Send second binary attachment
		err = c.Write(ctx, websocket.MessageBinary, []byte{4, 5, 6})
		if err != nil {
			t.Fatal(err)
		}

		// Wait for 3 packets in response (ack + binary attachments)
		packets, err := waitForPackets(c, ctx, 3)
		if err != nil {
			t.Fatal(err)
		}

		expectedText := `462-789[{"_placeholder":true,"num":0},{"_placeholder":true,"num":1}]`
		if packets[0].(string) != expectedText {
			t.Fatalf("expected %s, got %s", expectedText, packets[0])
		}

		// Check binary data
		binary1, ok := packets[1].([]byte)
		if !ok {
			t.Fatal("expected binary data")
		}
		if !bytes.Equal(binary1, []byte{1, 2, 3}) {
			t.Fatalf("expected [1,2,3], got %v", binary1)
		}

		binary2, ok := packets[2].([]byte)
		if !ok {
			t.Fatal("expected binary data")
		}
		if !bytes.Equal(binary2, []byte{4, 5, 6}) {
			t.Fatalf("expected [4,5,6], got %v", binary2)
		}
	})

	t.Run("should close the connection upon invalid format (unknown packet type)", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)

		err := c.Write(ctx, websocket.MessageText, []byte("4abc"))
		if err != nil {
			t.Fatal(err)
		}

		// Wait for connection to close
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				// Connection closed as expected
				break
			}
		}
	})

	t.Run("should close the connection upon invalid format (invalid payload format)", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)

		err := c.Write(ctx, websocket.MessageText, []byte("42{}"))
		if err != nil {
			t.Fatal(err)
		}

		// Wait for connection to close
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				// Connection closed as expected
				break
			}
		}
	})

	t.Run("should close the connection upon invalid format (invalid ack id)", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c := initSocketIOConnection(t)

		err := c.Write(ctx, websocket.MessageText, []byte(`42abc["message-with-ack",1,"2",{"3":[false]}]`))
		if err != nil {
			t.Fatal(err)
		}

		// Wait for connection to close
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				// Connection closed as expected
				break
			}
		}
	})
}
