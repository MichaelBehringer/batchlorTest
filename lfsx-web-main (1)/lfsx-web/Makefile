# Import environment settings from .env file
cnf ?= config.env
include $(cnf)
export $(shell sed 's/=.*//' $(cnf))

# Import deploy configuration
dpl ?= deploy.env
include $(dpl)
export $(shell sed 's/=.*//' $(dpl))

# Get the current version
VERSION=$(shell cat ./VERSION)
SHELL=/bin/bash

.PHONY: help

# Output help for every task
help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
.DEFAULT_GOAL := help

build: build-contoller build-lfs ## Build both the lfs and controller container

build-nc: build-lfs-nc build-controller-nc ## Build both the lfs and controller container without cache

build-lfs: ## Build the lfs container
	set -e ;\
	lfsLastCommitHash=$$(curl -s "https://notNeeded:"$$(cat $(GIT_API_KEY))"@gitea.hama.de/api/v1/repos/LFS/hama.java.lfs/commits?sha="${LFS_RELEASE_BRANCH}"&limit=1" | jq -r .[0].sha); \
	lfsLastCommitHash="222"
	buildah bud --layers --network host \
		--secret id=giteaApiKey,src=$(GIT_API_KEY) --secret id=giteaSshKey,src=$(GIT_SSH_KEY) \
		--build-arg lfsLastCommitHash=$$lfsLastCommitHash \
		--build-arg LFS_RELEASE=52.0.0 \
		--build-arg LFS_RELEASE_BRANCH=${LFS_RELEASE_BRANCH} \
		--tag=hama.de/lfsx-web-lfs:v$(VERSION)-dev -f lfs/docker/Dockerfile . 
build-controller: ## Build the controller for the lfs container
	buildah bud --layers --network host --secret id=giteaSshKey,src=$(GIT_SSH_KEY) --build-arg VERSION="$(VERSION)" --tag=hama.de/lfsx-web-controller:v$(VERSION)-dev -f controller/docker/Dockerfile . 

build-lfs-nc: ## Build the lfs container without cache
	buildah bud --no-cache --layers --network host --secret id=giteaApiKey,src=$(GIT_API_KEY) --secret id=giteaSshKey,src=$(GIT_SSH_KEY) --tag=hama.de/lfsx-web-lfs:v$(VERSION)-dev -f lfs/docker/Dockerfile . 
build-controller-nc: ## Build the controller for the lfs container without cache
	buildah bud --no-cache --layers --network host --secret id=giteaSshKey,src=$(GIT_SSH_KEY) --build-arg VERSION="$(VERSION)" --tag=hama.de/lfsx-web-controller:v$(VERSION)-dev -f controller/docker/Dockerfile .

run-controller:  ## Run the container with the controller
	@ make stop-controller > /dev/null 2>&1 || true
	podman run -it --env-file=./config.env --network host --name lfsx-web-controller -u 1111:1001 --cap-drop ALL -p 5910:5910 hama.de/lfsx-web-controller:v$(VERSION)-dev
stop-controller: ## Stop an remove a previous started container running the LFSX
	podman stop lfsx-web-controller; podman rm lfsx-web-controller

run-lfs:  ## Run the container with the LFSX
	@ make stop-lfs > /dev/null 2>&1 || true
	podman run -it --env-file=./config.env --network host --name lfsx-web-lfs -u 1111:1001 --cap-drop ALL -p 5910:5910 hama.de/lfsx-web-lfs:v$(VERSION)-dev
stop-lfs: ## Stop an remove a previous started container running the LFSX
	podman stop lfsx-web-lfs; podman rm lfsx-web-lfs

release: build-nc publish ## Make a release by building and publishing all containers to the registry (version and latest)

publish: repo-login publish-latest publish-version ## Publishes all previously build containers to the image registry

publish-latest: tag-latest ## Publish the containers to the image resitry with the tag `latest`
	@echo 'Publish latest to $(DOCKER_REPO)'
	podman push $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-controller:latest
	podman push $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-lfs:latest

publish-version: tag-version ## Publish the containers to the image registry with the tag `{version}`
	@echo 'publish $(VERSION) to $(DOCKER_REPO)'
	podman push $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-controller:v$(VERSION)
	podman push $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-lfs:v$(VERSION)


tag: tag-latest tag-version ## Generate container tags for the `{version}` and `latest` tags

tag-latest: ## Generate container with `{version}` tag
	@echo 'Create tag latest'
	podman tag hama.de/lfsx-web-controller:v$(VERSION)-dev $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-controller:latest
	podman tag hama.de/lfsx-web-lfs:v$(VERSION)-dev $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-lfs:latest

tag-version: ## Generate container with `latest` tag
	@echo 'Create tag $(VERSION)'
	podman tag hama.de/lfsx-web-controller:v$(VERSION)-dev $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-controller:v$(VERSION)
	podman tag hama.de/lfsx-web-lfs:v$(VERSION)-dev $(DOCKER_REPO)/$(DOCKER_BASE_PATH)lfsx-web-lfs:v$(VERSION)

repo-login: ## Login to the remote repository
	podman login $(DOCKER_REPO)

clear-images: ## Remove all previously build images and all intermediate images created by this makefile
	podman rmi $$(podman images -a | grep -e '<none>' -e '\/lfsx-web-.*' | awk '{ print $3 }') -f

copy-theme: ## Copies and applies the GTK theme for the host
	sudo mkdir -p /usr/share/themes/Adwaita-Own/gtk-3.0/
	sudo cp lfs/docker/configs/gtk.css /usr/share/themes/Adwaita-Own/gtk-3.0/gtk.css
	gsettings set org.gnome.desktop.interface gtk-theme 'Adwaita'
	gsettings set org.gnome.desktop.interface gtk-theme 'Adwaita-Own'