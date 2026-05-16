##
## FLICK PROJECT, 2026
## flick/Makefile
## File description:
## MAKEFILE
##

COMPOSE       := docker compose
COMPOSE_DEV   := $(COMPOSE) -f docker-compose.yaml -f docker-compose.dev.yaml
COMPOSE_BUILD := $(COMPOSE) -f docker-compose.yaml -f docker-compose.build.yaml

### Show help message
help:
	@awk 'BEGIN {FS=":"} /^### / {desc=substr($$0,5); next} /^[a-zA-Z_-]+:/ && desc {printf "  \033[36m%-14s\033[0m %s\n", $$1, desc; desc=""} !/^### / {desc=""}' $(MAKEFILE_LIST)

### Build CLI binaries for all platforms (scripts/build.sh)
build:
	./scripts/build.sh

### Build CLI binaries then start the dev stack (web hot reload, api rebuilt)
dev: build
	$(COMPOSE_DEV) up --build

### Stop and remove the stack (works for dev or prod)
down:
	$(COMPOSE_DEV) down

### Start the prod stack from registry image
up:
	$(COMPOSE) up -d

### Pull latest images
pull:
	$(COMPOSE) pull

### Build prod docker images locally
images:
	$(COMPOSE_BUILD) build

### Build and push prod images to the registry
images-push: images
	$(COMPOSE_BUILD) push

### Remove build artifacts
clean:
	rm -rf build/bin tmp

.PHONY: help dev down up pull build images images-push clean
