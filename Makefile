##
## FLICK PROJECT, 2026
## flick/Makefile
## File description:
## MAKEFILE
##

COMPOSE       := docker compose
COMPOSE_DEV   := $(COMPOSE) -f docker-compose.yaml -f docker-compose.dev.yaml
COMPOSE_BUILD := $(COMPOSE) -f docker-compose.yaml -f docker-compose.build.yaml
MIGRATE       := $(COMPOSE) run --rm --no-deps flick-migrate

### Show help message
help:
	@awk 'BEGIN {FS=":"} /^### / {desc=substr($$0,5); next} /^[a-zA-Z_-]+:/ && desc {printf "  \033[36m%-14s\033[0m %s\n", $$1, desc; desc=""} !/^### / {desc=""}' $(MAKEFILE_LIST)

### Build CLI binaries for all platforms (scripts/build.sh)
build:
	./scripts/build.sh

### Build CLI binaries then start the dev stack (web hot reload, api rebuilt)
dev: build
	$(COMPOSE_DEV) up --build

### Stop and remove the prod stack, keeps volumes (safe for data)
down:
	$(COMPOSE) down

### Stop the dev stack and wipe its volumes (node_modules, .next cache)
down-dev:
	$(COMPOSE_DEV) down -v

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

### Create a new migration file (usage: make migrate-new name=<NAME>)
migrate-new:
	$(COMPOSE) run --rm --no-deps flick-migrate new $(name)

### Apply all pending migrations
migrate-up:
	$(MIGRATE) up

### Roll back the last migration
migrate-down:
	$(MIGRATE) down

### Show the status of all migrations
migrate-status:
	$(MIGRATE) status

### Remove build artifacts
clean:
	rm -rf build/bin tmp

.PHONY: help dev down down-dev up pull build images images-push clean migrate-new migrate-up migrate-down migrate-status
