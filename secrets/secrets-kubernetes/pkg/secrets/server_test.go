package secrets

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type testHandler struct {
	getFunc func(ctx context.Context, r *Request) (*SecretValue, error)
}

func (t *testHandler) Get(ctx context.Context, r *Request) (*SecretValue, error) {
	return t.getFunc(ctx, r)
}

func natsConnectionForTest(t *testing.T) *nats.Conn {
	t.Helper()

	s := natsserver.RunRandClientPortServer()
	t.Cleanup(s.Shutdown)

	nc, err := nats.Connect(s.ClientURL())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(nc.Close)

	return nc
}

func keyPairForTest(t *testing.T) nkeys.KeyPair {
	t.Helper()

	kp, err := nkeys.CreateCurveKeys()
	if err != nil {
		t.Fatal(err)
	}

	return kp
}

func TestNewServer(t *testing.T) {
	if _, err := NewServer("", nil, nil); err == nil {
		t.Errorf("server name shouldn't be blank")
	}

	if _, err := NewServer("kube", nil, nil); err == nil {
		t.Errorf("nats client shouldn't be blank")
	}

	nc := natsConnectionForTest(t)

	if _, err := NewServer("kube", nc, nil); err == nil {
		t.Errorf("handler shouldn't be blank")
	}

	handler := &testHandler{}

	if _, err := NewServer("kube", nc, handler); err == nil {
		t.Errorf("keypair shouldn't be blank")
	}

	kp := keyPairForTest(t)

	if _, err := NewServer("kube", nc, handler, WithKeyPair(kp)); err != nil {
		t.Error(err)
	}

	if _, err := NewServer("kube", nc, handler, WithEphemeralKey()); err != nil {
		t.Error(err)
	}
}

func TestServerLoop(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		nc := natsConnectionForTest(t)

		var handler testHandler

		server, err := NewServer("kube", nc, &handler, WithEphemeralKey())
		if err != nil {
			t.Fatal(err)
		}

		if err := server.Run(); err != nil {
			t.Error(err)
		}

		if err := server.Shutdown(false); err != nil {
			t.Error(err)
		}
	})

	t.Run("Drain", func(t *testing.T) {
		nc := natsConnectionForTest(t)

		var handler testHandler

		server, err := NewServer("kube", nc, &handler, WithEphemeralKey())
		if err != nil {
			t.Fatal(err)
		}

		if err := server.Run(); err != nil {
			t.Error(err)
		}

		if err := server.Shutdown(true); err != nil {
			t.Error(err)
		}
	})

	t.Run("BrokenNats", func(t *testing.T) {
		nc := natsConnectionForTest(t)

		var handler testHandler

		server, err := NewServer("kube", nc, &handler, WithEphemeralKey())
		if err != nil {
			t.Fatal(err)
		}

		// disconnect nats
		nc.Close()

		if err := server.Run(); err == nil {
			t.Error("nats should be connected")
		}
	})
}

func TestServerXkey(t *testing.T) {
	nc := natsConnectionForTest(t)

	var handler testHandler

	server, err := NewServer("kube", nc, &handler, WithEphemeralKey())
	if err != nil {
		t.Fatal(err)
	}

	serverPubKey, err := server.key.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	if err := server.Run(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Shutdown(false) })

	rawReply, err := nc.Request(server.subjectMapper.SecretsSubject()+".server_xkey", nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := serverPubKey, string(rawReply.Data); want != got {
		t.Errorf("wanted %+v, got %+v", want, got)
	}
}

func TestServerGet(t *testing.T) {
	nc := natsConnectionForTest(t)

	basicGetFunc := func(ctx context.Context, r *Request) (*SecretValue, error) {
		return &SecretValue{
			StringSecret: "value",
		}, nil
	}

	basicCheckResponse := func(t *testing.T, r Response) {
		if r.Secret.StringSecret != "value" {
			t.Fatal("secret value mismatch")
		}
		if r.Error != nil {
			t.Fatal("didnt expect an error here")
		}
	}

	handler := &testHandler{}

	server, err := NewServer("kube", nc, handler, WithEphemeralKey())
	if err != nil {
		t.Fatal(err)
	}

	serverPubKey, err := server.key.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	if err := server.Run(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Shutdown(false) })

	kp := keyPairForTest(t)
	hostPubKey, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	reqCtx := Context{
		Application: &ApplicationContext{
			Name: "appname",
		},
		EntityJwt: "eyJ0eXAiOiJqd3QiLCJhbGciOiJFZDI1NTE5In0.eyJqdGkiOiJxdmVOakZjcW51dWhQaVJUMkU1YWJXIiwiaWF0IjoxNzIxODM0ODg5LCJpc3MiOiJBQk9HQjRXNURPWDNVTzNSVldXUUdZU01WWEhSUFFZWFZaUDVVNFZGTUpEQ1lDV0FSN1M1Q1lNTyIsInN1YiI6Ik1DNUNDNFVENUxQRFo0QzdaTkFFQTRPWlEzQkVGTFNWUTc0MlczVEVUM09OS1M0RFJCVk5NNUlDIiwid2FzY2FwIjp7Im5hbWUiOiJodHRwLWhlbGxvLXdvcmxkIiwiaGFzaCI6IkNFOTAxOTJDOTlDMEIyQzYwOEIyRTJDQjYxOUE5MjUxRkI2ODE4NTZDMTU2ODFCMUJDRDYyRUVEQTJENTEyOEUiLCJ0YWdzIjpbIndhc21jbG91ZC5jb20vZXhwZXJpbWVudGFsIl0sInJldiI6MCwidmVyIjoiMC4xLjAiLCJwcm92IjpmYWxzZX0sIndhc2NhcF9yZXZpc2lvbiI6M30.8awbkvrBnRKLpz88s7GXYCW0onpKf_nNfsj7pXhCyvq8pm4y2IotrIPCdBvWqDvDouX4VAM6DQQUHuI-VdKYAA",
		HostJwt:   "eyJ0eXAiOiJqd3QiLCJhbGciOiJFZDI1NTE5In0.eyJqdGkiOiJuTGdta2Zud2p2Nkw1R28xSlNUdU0zIiwiaWF0IjoxNzIyMDE5OTk1LCJpc3MiOiJBQzNGU0IzT0VSQ1IzVU00WVNWUjJUQURFVlFWUTNITVpQQUtHS082QkNRSTRSNEFITFY2SVhSMiIsInN1YiI6Ik5ETlBUM0QzWVNUQzVKR0g2QVBKUDZBTVZYUVk2QklETVVXWkdTU1FXMjZWSjNINFBDRjJTU0ZSIiwid2FzY2FwIjp7Im5hbWUiOiJkZWxpY2F0ZS1icmVlemUtOTc4NSIsImxhYmVscyI6eyJzZWxmX3NpZ25lZCI6InRydWUifX0sIndhc2NhcF9yZXZpc2lvbiI6M30.5LM_GOpo-6qg0kDrIP_jswI_ZQfOILzHT-FHixvUeAf-1isamLg81S-rb84w6topfvevI6quyV3b-uHZt6q9BQ",
	}

	tests := map[string]struct {
		plainText     bool
		req           Request
		protocolError bool
		hostKey       string
		getFunc       func(ctx context.Context, r *Request) (*SecretValue, error)
		checkResponse func(*testing.T, Response)
	}{
		"blank": {
			plainText:     true,
			protocolError: true,
		},
		"happyPath": {
			req: Request{
				Key:     "secret",
				Context: reqCtx,
			},
		},
		"upstreamError": {
			req: Request{
				Key:     "secret",
				Context: reqCtx,
			},
			protocolError: true,
			getFunc: func(context.Context, *Request) (*SecretValue, error) {
				return nil, ErrUpstream.With("boom")
			},
			checkResponse: func(t *testing.T, resp Response) {
				if want, got := ErrUpstream.Error(), resp.Error.Error(); want != got {
					t.Errorf("want %v, got %v", want, got)
				}
			},
		},
		"badSecret": {
			req: Request{
				Key:     "secret",
				Context: reqCtx,
			},
			hostKey:       "badkey",
			protocolError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.getFunc != nil {
				handler.getFunc = test.getFunc
			} else {
				handler.getFunc = basicGetFunc
			}
			rawData, err := json.Marshal(&test.req)
			if err != nil {
				t.Fatal(err)
			}

			rawReq := nats.NewMsg(server.subjectMapper.SecretsSubject() + ".get")

			if !test.plainText {
				sealedData, err := kp.Seal(rawData, serverPubKey)
				if err != nil {
					t.Fatal(err)
				}

				rawReq.Data = sealedData
				hostKey := hostPubKey
				if test.hostKey != "" {
					hostKey = test.hostKey
				}
				rawReq.Header.Add(WasmCloudHostXkey, hostKey)
			}

			rawReply, err := nc.RequestMsg(rawReq, time.Second)
			if err != nil {
				t.Fatal(err)
			}

			var resp Response

			// the presence of the response header indicates if this is an encrypted response or not
			// plain responses are protocol errors
			responseKey := rawReply.Header.Get(WasmCloudResponseXkey)
			if test.protocolError {
				if responseKey != "" {
					t.Error("saw encryption header on protocol error")
				}

				if err := json.Unmarshal(rawReply.Data, &resp); err != nil {
					t.Fatal(err)
				}

				if resp.Error == nil {
					t.Fatal("Expected an error but got none")
				}

				if test.checkResponse != nil {
					test.checkResponse(t, resp)
				}
				return
			}

			if !test.protocolError && responseKey == "" {
				t.Error("missing encryption header")
			}

			rawResponse, err := kp.Open(rawReply.Data, responseKey)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(rawResponse, &resp); err != nil {
				t.Fatal(err)
			}

			if test.checkResponse != nil {
				test.checkResponse(t, resp)
			} else {
				basicCheckResponse(t, resp)
			}
		})
	}
}
