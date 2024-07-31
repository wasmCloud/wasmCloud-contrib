package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
)

const (
	DefaultSecretsBusPrefix       = "wasmcloud.secrets"
	DefaultSecretsProtocolVersion = "v1alpha1"
	WasmCloudHostXkey             = "WasmCloud-Host-Xkey"
	WasmCloudResponseXkey         = "Server-Response-Xkey"
)

var (
	ErrInvalidServerConfig = errors.New("invalid server configuration")

	ErrSecretNotFound = newResponseError("SecretNotFound", false)
	ErrInvalidRequest = newResponseError("InvalidRequest", false)
	ErrInvalidHeaders = newResponseError("InvalidHeaders", false)
	ErrInvalidPayload = newResponseError("InvalidPayload", false)
	ErrEncryption     = newResponseError("EncryptionError", false)
	ErrDecryption     = newResponseError("DecryptionError", false)

	ErrInvalidEntityJWT = newResponseError("InvalidEntityJWT", true)
	ErrInvalidHostJWT   = newResponseError("InvalidHostJWT", true)
	ErrUpstream         = newResponseError("UpstreamError", true)
	ErrPolicy           = newResponseError("PolicyError", true)
	ErrOther            = newResponseError("Other", true)
)

type ResponseError struct {
	Tip        string
	HasMessage bool
	Message    string
}

func (re ResponseError) With(msg string) *ResponseError {
	otherError := re
	otherError.Message = msg
	return &otherError
}

func (re ResponseError) Error() string {
	return re.Tip
}

func (re *ResponseError) UnmarshalJSON(data []byte) error {
	serdeSpecial := make(map[string]string)
	if err := json.Unmarshal(data, &serdeSpecial); err != nil {
		var msg string
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		*re = *ErrOther.With(msg)
		return nil
	}
	if len(serdeSpecial) != 1 {
		return errors.New("couldn't parse ResponseError")
	}
	for k, v := range serdeSpecial {
		*re = ResponseError{Tip: k, HasMessage: v != "", Message: v}
		break
	}

	return nil
}

func (re *ResponseError) MarshalJSON() ([]byte, error) {
	if re == nil {
		return nil, nil
	}

	if !re.HasMessage {
		return json.Marshal(re.Tip)
	}

	serdeSpecial := make(map[string]string)
	serdeSpecial[re.Tip] = re.Message

	return json.Marshal(serdeSpecial)
}

func newResponseError(tip string, hasMessage bool) *ResponseError {
	return &ResponseError{Tip: tip, HasMessage: hasMessage}
}

// SubjectMapper helps manipulating NATS subjects
type SubjectMapper struct {
	Prefix      string
	Version     string
	ServiceName string
}

func (s SubjectMapper) QueueGroupName() string {
	return fmt.Sprintf("%s.%s", s.Prefix, s.ServiceName)
}

func (s SubjectMapper) SecretsSubject() string {
	return fmt.Sprintf("%s.%s.%s", s.Prefix, s.Version, s.ServiceName)
}

func (s SubjectMapper) SecretWildcardSubject() string {
	return fmt.Sprintf("%s.>", s.SecretsSubject())
}

func (s SubjectMapper) ParseOperation(subject string) string {
	prefix := s.SecretsSubject() + "."
	if !strings.HasPrefix(subject, prefix) {
		return ""
	}

	return strings.TrimPrefix(subject, prefix)
}

type ApplicationContext struct {
	Policy string `json:"policy"`
	Name   string `json:"name"`
}

type applicationContextPolicy struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

func (a ApplicationContext) PolicyProperties() (json.RawMessage, error) {
	policy := &applicationContextPolicy{}
	err := json.Unmarshal([]byte(a.Policy), policy)
	return policy.Properties, err
}

type Context struct {
	/// The application the entity belongs to.
	/// TODO: should this also be a JWT, but signed by the host?
	Application *ApplicationContext `json:"application,omitempty"`
	/// The component or provider's signed JWT.
	EntityJwt string `json:"entity_jwt"`
	/// The host's signed JWT.
	HostJwt string `json:"host_jwt"`
}

func (ctx Context) IsValid() *ResponseError {
	if _, _, err := ctx.EntityCapabilities(); err != nil {
		return err
	}

	if _, _, err := ctx.HostCapabilities(); err != nil {
		return err
	}

	return nil
}

func (ctx Context) EntityCapabilities() (*WasCap, *ComponentClaims, *ResponseError) {
	token, err := jwt.ParseWithClaims(ctx.EntityJwt, &WasCap{}, KeyPairFromIssuer())
	if err != nil {
		return nil, nil, ErrInvalidEntityJWT.With(err.Error())
	}

	wasCap, ok := token.Claims.(*WasCap)
	if !ok {
		return nil, nil, ErrInvalidEntityJWT.With("not wascap")
	}

	compCap := &ComponentClaims{}
	if err := json.Unmarshal(wasCap.Was, compCap); err != nil {
		return nil, nil, ErrInvalidEntityJWT.With(err.Error())
	}

	return wasCap, compCap, nil
}

func (ctx Context) HostCapabilities() (*WasCap, *HostClaims, *ResponseError) {
	token, err := jwt.ParseWithClaims(ctx.HostJwt, &WasCap{}, KeyPairFromIssuer())
	if err != nil {
		return nil, nil, ErrInvalidHostJWT.With(err.Error())
	}

	wasCap, ok := token.Claims.(*WasCap)
	if !ok {
		return nil, nil, ErrInvalidHostJWT.With("not wascap")
	}

	hostCap := &HostClaims{}
	if err := json.Unmarshal(wasCap.Was, hostCap); err != nil {
		return nil, nil, ErrInvalidHostJWT.With(err.Error())
	}

	return wasCap, hostCap, nil
}

type Request struct {
	Key     string  `json:"key"`
	Field   string  `json:"field"`
	Version string  `json:"version"`
	Context Context `json:"context"`
}

func (s Request) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(&s)
}

func (s Request) String() string {
	var b bytes.Buffer
	_ = s.Write(&b)
	return b.String()
}

type ByteArray []uint8

func (u ByteArray) MarshalJSON() ([]byte, error) {
	var result string
	if u == nil {
		return nil, nil
	}

	result = strings.Join(strings.Fields(fmt.Sprintf("%d", u)), ",")
	return []byte(result), nil
}

type SecretValue struct {
	Version      string    `json:"version,omitempty"`
	StringSecret string    `json:"string_secret,omitempty"`
	BinarySecret ByteArray `json:"binary_secret,omitempty"`
}

type Response struct {
	Secret *SecretValue   `json:"secret,omitempty"`
	Error  *ResponseError `json:"error,omitempty"`
}

func (r Response) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(&r)
}

func (r Response) String() string {
	var b bytes.Buffer
	_ = r.Write(&b)
	return b.String()
}

type Handler interface {
	Get(ctx context.Context, r *Request) (*SecretValue, error)
}

type ComponentClaims struct {
	/// A descriptive name for this component, should not include version information or public key
	Name string `json:"name"`
	/// A hash of the module's bytes as they exist without the embedded signature. This is stored so wascap
	/// can determine if a WebAssembly module's bytecode has been altered after it was signed
	ModuleHash string `json:"hash"`

	/// List of arbitrary string tags associated with the claims
	Tags []string `json:"tags"`

	/// Indicates a monotonically increasing revision number.  Optional.
	Rev int32 `json:"rev"`

	/// Indicates a human-friendly version string
	Ver string `json:"ver"`

	/// An optional, code-friendly alias that can be used instead of a public key or
	/// OCI reference for invocations
	CallAlias string `json:"call_alias"`

	/// Indicates whether this module is a capability provider
	Provider bool `json:"prov"`

	jwt.RegisteredClaims
}

type CapabilityProviderClaims struct {
	/// A descriptive name for the capability provider
	Name string
	/// A human-readable string identifying the vendor of this provider (e.g. Redis or Cassandra or NATS etc)
	Vendor string
	/// Indicates a monotonically increasing revision number.  Optional.
	Rev int32
	/// Indicates a human-friendly version string. Optional.
	Ver string
	/// If the provider chooses, it can supply a JSON schma that describes its expected link configuration
	ConfigSchema json.RawMessage
	/// The file hashes that correspond to the achitecture-OS target triples for this provider.
	TargetHashes map[string]string
}

type HostClaims struct {
	/// Optional friendly descriptive name for the host
	Name string
	/// Optional labels for the host
	Labels map[string]string
}

type WasCap struct {
	jwt.RegisteredClaims

	/// Custom jwt claims in the `wascap` namespace
	Was json.RawMessage `json:"wascap,omitempty"`

	/// Internal revision number used to aid in parsing and validating claims
	Revision int32 `json:"wascap_revision,omitempty"`
}

func (w WasCap) ParseCapability(dst interface{}) error {
	return json.Unmarshal(w.Was, dst)
}
