package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type (
	ServerErrorCallback  func(msg *nats.Msg, err error)
	ServerContextCreator func() context.Context
)

type Server struct {
	queue         *nats.Subscription
	natsConn      *nats.Conn
	handler       Handler
	onError       ServerErrorCallback
	key           nkeys.KeyPair
	pubKey        string
	subjectMapper SubjectMapper
	ctxCreator    ServerContextCreator
}

type ServerOption func(*Server) error

func WithEphemeralKey() ServerOption {
	return func(s *Server) error {
		var err error
		s.key, err = nkeys.CreateCurveKeys()
		return err
	}
}

func WithKeyPair(kp nkeys.KeyPair) ServerOption {
	return func(s *Server) error {
		s.key = kp
		return nil
	}
}

func WithSubjectMapper(m SubjectMapper) ServerOption {
	return func(s *Server) error {
		s.subjectMapper = m
		return nil
	}
}

func WithErrorCallback(cb ServerErrorCallback) ServerOption {
	return func(s *Server) error {
		s.onError = cb
		return nil
	}
}

func WithRequestContext(cb ServerContextCreator) ServerOption {
	return func(s *Server) error {
		s.ctxCreator = cb
		return nil
	}
}

func NewServer(name string, nc *nats.Conn, handler Handler, opts ...ServerOption) (*Server, error) {
	server := &Server{
		natsConn:   nc,
		handler:    handler,
		onError:    func(*nats.Msg, error) {},
		ctxCreator: func() context.Context { return context.Background() },
		subjectMapper: SubjectMapper{
			Version:     DefaultSecretsProtocolVersion,
			Prefix:      DefaultSecretsBusPrefix,
			ServiceName: name,
		},
	}

	for _, opt := range opts {
		if err := opt(server); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidServerConfig, err)
		}
	}

	if name == "" {
		return nil, fmt.Errorf("%w: missing name", ErrInvalidServerConfig)
	}

	if server.natsConn == nil {
		return nil, fmt.Errorf("%w: nats connection", ErrInvalidServerConfig)
	}

	if server.handler == nil {
		return nil, fmt.Errorf("%w: missing handler", ErrInvalidServerConfig)
	}

	if server.key == nil {
		return nil, fmt.Errorf("%w: missing key pair", ErrInvalidServerConfig)
	}

	if server.ctxCreator == nil {
		return nil, fmt.Errorf("%w: context creator", ErrInvalidServerConfig)
	}

	var err error
	server.pubKey, err = server.key.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidServerConfig, err)
	}

	return server, nil
}

func (s *Server) Process(ctx context.Context, msg *nats.Msg) {
	nakCallback := func(respErr *ResponseError) {
		s.onError(msg, respErr)

		resp := Response{Error: respErr}

		data, err := json.Marshal(&resp)
		if err != nil {
			s.onError(msg, err)
			return
		}

		if err := msg.Respond(data); err != nil {
			s.onError(msg, err)
		}
	}

	operation := s.subjectMapper.ParseOperation(msg.Subject)
	switch operation {
	case "get":
		hostPubKey := msg.Header.Get(WasmCloudHostXkey)
		if hostPubKey == "" {
			nakCallback(ErrInvalidHeaders)
			return
		}

		rawReq, err := s.key.Open(msg.Data, hostPubKey)
		if err != nil {
			nakCallback(ErrDecryption)
			return
		}

		req := &Request{}
		if err := json.Unmarshal(rawReq, &req); err != nil {
			nakCallback(ErrInvalidPayload)
			return
		}

		if err := req.Context.IsValid(); err != nil {
			nakCallback(err)
			return
		}

		secretValue, err := s.handler.Get(ctx, req)
		if err != nil {
			if respErr, ok := err.(*ResponseError); ok {
				nakCallback(respErr)
			} else {
				nakCallback(ErrUpstream.With(err.Error()))
			}
			return
		}

		responseKey, err := nkeys.CreateCurveKeys()
		if err != nil {
			nakCallback(ErrEncryption)
			return
		}
		ephemeralPubKey, err := responseKey.PublicKey()
		if err != nil {
			nakCallback(ErrEncryption)
			return
		}

		data, err := json.Marshal(&Response{Secret: secretValue})
		if err != nil {
			nakCallback(ErrInvalidPayload)
			return
		}

		respMsg := nats.NewMsg("")
		respMsg.Header.Add(WasmCloudResponseXkey, ephemeralPubKey)

		respMsg.Data, err = responseKey.Seal(data, hostPubKey)
		if err != nil {
			nakCallback(ErrEncryption)
			return
		}

		if err := msg.RespondMsg(respMsg); err != nil {
			nakCallback(ErrOther.With("failed to respond 'get'"))
		}
	case "server_xkey":
		if err := msg.Respond([]byte(s.pubKey)); err != nil {
			nakCallback(ErrInvalidRequest)
		}

	default:
		nakCallback(ErrInvalidRequest)
	}
}

func (s *Server) Run() error {
	var queueErr error

	s.queue, queueErr = s.natsConn.QueueSubscribe(
		s.subjectMapper.SecretWildcardSubject(),
		s.subjectMapper.QueueGroupName(),
		func(msg *nats.Msg) {
			s.Process(s.ctxCreator(), msg)
		})

	return queueErr
}

func (s *Server) Shutdown(shouldDrain bool) error {
	if s.queue != nil {
		if shouldDrain {
			return s.queue.Drain()
		} else {
			return s.queue.Unsubscribe()
		}
	}

	return nil
}
