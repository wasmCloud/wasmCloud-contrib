VERSION?=dev
IMG?=ghcr.io/wasmcloud/contrib/secrets-kubernetes:$(VERSION)

build:
	docker build -t $(IMG) $(PWD)

dev-init:
	kubectl apply -k deploy/dev
dev-deploy: build
	kubectl -n wasmcloud-secrets rollout restart deployment --selector=app=wasmcloud-secrets
dev-logs:
	while true; do kubectl -n wasmcloud-secrets logs -f -l=app=wasmcloud-secrets; sleep 1; done

