VERSION ?= latest
NAME_PREFIX ?= nof-
NATS_APP_NAME ?= $(NAME_PREFIX)nats
NATS_SURVEYOR_APP_NAME ?= $(NAME_PREFIX)nats-surveyor
CLIENT_APP_NAME ?= $(NAME_PREFIX)client
SERVER_APP_NAME ?= $(NAME_PREFIX)server
GRAFANA_NEED_VOLUME ?=
GRAFANA_APP_NAME ?= $(NAME_PREFIX)grafana
PROMETHEUS_API_TOKEN ?=
CLIENT_COUNT ?= 1

docker: nof-client nof-server

deploy: deploy-nats deploy-nats-surveyor deploy-server

nof-client:
	docker build --platform=linux/amd64 --tag nof-client:$(VERSION) . --file Dockerfile.client
	[[ ! -z "${DOCKER_REPO_BASE}" ]] && \
		docker tag nof-client:$(VERSION) ${DOCKER_REPO_BASE}nof-client:$(VERSION) && \
		docker push ${DOCKER_REPO_BASE}nof-client:$(VERSION) || true

nof-server:
	docker build --platform=linux/amd64 --tag nof-server:$(VERSION) . --file Dockerfile.server
	[[ ! -z "${DOCKER_REPO_BASE}" ]] && \
		docker tag nof-server:$(VERSION) ${DOCKER_REPO_BASE}nof-server:$(VERSION) && \
		docker push ${DOCKER_REPO_BASE}nof-server:$(VERSION) || true

deploy-nats:
	cd deployment/nats && \
		fly deploy --app $(NATS_APP_NAME)

deploy-nats-surveyor:
	fly secrets set --app $(NATS_SURVEYOR_APP_NAME) NATS_ADDR="$(NATS_APP_NAME).internal:4222"
	cd deployment/nats-surveyor && \
		fly deploy --app $(NATS_SURVEYOR_APP_NAME)

deploy-server:
	fly secrets set --app $(SERVER_APP_NAME) NATS_ADDR="$(NATS_APP_NAME).internal:4222"
	fly deploy --config deployment/server/fly.toml --app $(SERVER_APP_NAME)

deploy-grafana:
ifneq ($(PROMETHEUS_API_TOKEN), )
	fly secrets set --app $(GRAFANA_APP_NAME) PROMETHEUS_API_TOKEN=$(PROMETHEUS_API_TOKEN)
endif
ifneq ($(GRAFANA_NEED_VOLUME), )
	fly volumes create grafana_storage --app $(GRAFANA_APP_NAME) --region ord
endif
	cd deployment/grafana && \
		fly deploy --app $(GRAFANA_APP_NAME)

launch-client:
	go run ./cmd/simulate launch --app $(CLIENT_APP_NAME) --env NATS_ADDR="$(NATS_APP_NAME).internal:4222" --env CLIENT_COUNT="$(CLIENT_COUNT)"

remove-all-clients:
	go run ./cmd/simulate remove-all --app $(CLIENT_APP_NAME)

open-server:
	fly open --app $(SERVER_APP_NAME) /?with_responses