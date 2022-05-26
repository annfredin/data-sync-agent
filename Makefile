# variables
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test ./lib/...
GOGET=$(GOCMD) get -u -v
PROTOC    = protoc

# build variables
KUBE_NAMESPABE = iot-data-transfer
BASE_REPO = gitlab.com/gitlab-org
BASE_REPO_URL = data-sync-agent

REGISTRY = registry.gitlab.com/gitlab-org/data-sync-agent
BINARY = data-sync-agent
GOARCH = amd64
CHART_FOLDER = ./chart/data-sync-agent
CHART_VALUES_FOLDER = ./chart/data-sync-agent

# TAG =  $$(git describe --abbrev=0 --tags)
TAG =  2.0.1
DEPLOY_ENV = prod

LDFLAGS = -ldflags="$$(govvv -flags -version ${TAG})"

IMG    := ${REGISTRY}:${TAG}
IMGLATEST := ${REGISTRY}:latest

# commands...
# build commands

.PHONY: make clean
clean:
	@rm -rf ${BINARY}-linux-${GOARCH}
	@rm -rf ${BINARY}-darwin-${GOARCH}


.PHONY: all
all:clean osx linux

.PHONY: osx
osx:
	env CGO_ENABLED=0 GOOS=darwin GOARCH=${GOARCH} go build -a -installsuffix cgo ${LDFLAGS} -o ${BINARY}-darwin-${GOARCH} . ;

.PHONY: linux
linux:
	env CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build -a -installsuffix cgo ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} . ;

.PHONY: run
run: osx
	./${BINARY}-darwin-${GOARCH}


.PHONY: dockerbuild
dockerbuild:
	docker build -t ${IMG}-${DEPLOY_ENV} .   

.PHONY: dockerpush
dockerpush:
	docker push ${IMG}-${DEPLOY_ENV} 

# only for ci
.PHONY: dockerbuildci
dockerbuildci:
ifeq ($(CI),true)
ifeq ($(CI_COMMIT_REF_NAME),master)
	docker build -t ${IMG} .
	docker tag ${IMG} ${IMGLATEST}
else
	docker build -t ${IMG}-$(CI_COMMIT_REF_NAME) .
endif
endif

# only for ci
.PHONY: dockerpushci
dockerpushci:
ifeq ($(CI),true)
ifeq ($(CI_COMMIT_REF_NAME),master)
	docker push ${IMG}
	docker push ${IMGLATEST}
else
	docker push ${IMG}-$(CI_COMMIT_REF_NAME)
endif
endif

.PHONY: dockerci
dockerci: dockerbuildci dockerpushci

# only for ci
.PHONY: helminstallci
helminstallci:
ifeq ($(CI),true)
ifeq ($(CI_COMMIT_REF_NAME),master)
	helm upgrade --install ${BINARY}  --recreate-pods -f ${CHART_VALUES_FOLDER}/values.yaml --set image.tag=${TAG} --namespace ${KUBE_NAMESPABE} ${CHART_FOLDER}
else
	helm upgrade --install ${BINARY}  --recreate-pods -f ${CHART_VALUES_FOLDER}/values.yaml -f  --set image.tag=${TAG}-$(CI_COMMIT_REF_NAME) --namespace ${KUBE_NAMESPABE} ${CHART_FOLDER}
endif
	make createjobstatuscode
endif


.PHONY: helminstall
helminstall:
	kubectx ${DEPLOY_ENV}
	helm upgrade --install ${BINARY}  --recreate-pods -f ${CHART_VALUES_FOLDER}/values.yaml  --set image.tag=${TAG}-${DEPLOY_ENV} --namespace ${KUBE_NAMESPABE} ${CHART_FOLDER}

.PHONY: helmdeploy
helmdeploy: dockerbuild dockerpush helminstall


.PHONY: createjobstatuscode
createjobstatuscode:
	echo 1 > JOB_STATUS_CODE

.PHONY: removejobstatuscode
removejobstatuscode:
	@rm -rf JOB_STATUS_CODE

.PHONY: notifyci
notifyci: slacknotify removejobstatuscode

.PHONY: slacknotify
slacknotify:
	~/notify.sh

# git commands.

.PHONY: go-mod-vendor
go-mod-vendor:
	go mod init $(BASE_REPO_URL) && go mod vendor

.PHONY: pull-core
pull-core:
	go get -u ${BASE_REPO}/core && go mod vendor

.PHONY: pull-core-run
pull-core-run:	pull-core
	go run main.go
	
.PHONY: kill-port
kill-port:
	kill -9 `lsof -i TCP:$(port) | awk '/LISTEN/{print $2}'`

.PHONY: commit-history
commit-hostory:
	git log --format='%Cred%h%Creset %s %Cgreen(%cr) %C(blue)<%an>%Creset%C(yellow)%d%Creset'

.PHONY: git-commit-run
git-commit-run:
	git pull origin
	git add . 
	cz c
	cz bump

.PHONY: git-push-run
git-push-run:
	git push
	git push origin --tags

.PHONY: git-push
git-push: git-commit-run git-push-run

