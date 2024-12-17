package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/wasmCloud/contrib/secrets/secrets-kubernetes/pkg/secrets"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ServiceName = "kube"
)

type kubeSecretsServer struct{}

type kubeApplicationPolicy struct {
	Impersonate string `json:"impersonate"`
	Namespace   string `json:"namespace"`
}

func parseApplicationPolicy(r *secrets.Request) (*kubeApplicationPolicy, error) {
	rawPolicy, err := r.Context.Application.PolicyProperties()
	if err != nil {
		return nil, err
	}
	policy := &kubeApplicationPolicy{Namespace: "default"}
	err = json.Unmarshal(rawPolicy, policy)
	return policy, err
}

func (s *kubeSecretsServer) Get(ctx context.Context, r *secrets.Request) (*secrets.SecretValue, error) {
	policy, err := parseApplicationPolicy(r)
	if err != nil {
		return nil, secrets.ErrPolicy.With(err.Error())
	}
	slog.Info("Get", slog.String("application", r.Context.Application.Name), slog.String("impersonate", policy.Impersonate), slog.String("key", r.Key), slog.String("field", r.Field))

	if r.Key == "" {
		return nil, secrets.ErrOther.With("missing secret name")
	}

	if r.Field == "" {
		return nil, secrets.ErrOther.With("missing secret key/field")
	}

	kubeClient, err := kubeClientWithImpersonation(policy.Impersonate)
	if err != nil {
		return nil, secrets.ErrUpstream.With(err.Error())
	}

	kubeSecret, err := kubeClient.Secrets(policy.Namespace).Get(ctx, r.Key, metav1.GetOptions{})
	if err != nil {
		return nil, secrets.ErrUpstream.With(err.Error())
	}

	kubeEntryValue, ok := kubeSecret.Data[r.Field]
	if !ok {
		return nil, secrets.ErrSecretNotFound
	}

	return &secrets.SecretValue{
		StringSecret: string(kubeEntryValue),
		Version:      kubeSecret.ResourceVersion,
	}, nil
}

func kubeClientWithImpersonation(role string) (clientcorev1.CoreV1Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil).ClientConfig()
	if err != nil {
		return nil, err
	}

	if role != "" {
		config.Impersonate.UserName = role
	}

	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return kubeClientset.CoreV1(), nil
}

func main() {
	var (
		natsURL            = flag.String("nats-url", nats.DefaultURL, "Nats URL")
		natsCreds          = flag.String("nats-creds", "", "NATS credentials file path.")
		secretsBackendSeed = flag.String("backend-seed", "", "NKeys Curve Seed. Leave blank for ephemeral key, only recommended for development use")
	)
	flag.Parse()

	slog.Info("Starting", slog.String("nats-url", *natsURL))

	s := &kubeSecretsServer{}

	natsConnectOps := []nats.Option{}
	if *natsCreds != "" {
		natsConnectOps = append(natsConnectOps, nats.UserCredentials(*natsCreds))
	}

	nc, err := nats.Connect(*natsURL, natsConnectOps...)
	if err != nil {
		slog.Error("Couldn't setup nats client", slog.Any("error", err))
		os.Exit(1)
	}

	errorCallback := func(_ *nats.Msg, err error) {
		slog.Error("server error", slog.Any("error", err))
	}

	var secretsBackendKey nkeys.KeyPair

	if *secretsBackendSeed != "" {
		secretsBackendKey, err = nkeys.FromSeed([]byte(*secretsBackendSeed))
	} else {
		slog.Info("Creating ephemeral curve keys. DO NOT USE THIS IN PRODUCTION.")
		secretsBackendKey, err = nkeys.CreateCurveKeys()
	}
	if err != nil {
		slog.Error("Couldn't setup XKey", slog.Any("error", err))
		os.Exit(1)
	}

	secretsServer, err := secrets.NewServer(ServiceName,
		nc,
		s,
		secrets.WithKeyPair(secretsBackendKey),
		secrets.WithErrorCallback(errorCallback),
	)
	if err != nil {
		slog.Error("Couldn't setup secrets server", slog.Any("error", err))
		os.Exit(1)
	}

	mainCtx, mainCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGHUP)
	defer mainCancel()

	if err := secretsServer.Run(); err != nil {
		slog.Error("Couldn't setup secrets protocol server", slog.Any("error", err))
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-mainCtx.Done()
		slog.Info("Signal received. Draining...")
		if err := secretsServer.Shutdown(true); err != nil {
			slog.Error("Couldn't drain all messages", slog.Any("error", err))
		} else {
			slog.Info("Drained all messages")
		}
		wg.Done()
	}()

	slog.Info("Server is up")
	wg.Wait()
}
