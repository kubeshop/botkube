IMAGE_REPO=infracloud/botkube
TAG=$(shell cut -d'=' -f2- .release)

.DEFAULT_GOAL := build
.PHONY: release git-tag check-git-status build pre-build tag-image publish

#Docker Tasks
#Make a release
release: check-git-status build tag-image publish git-tag 
	@echo "Successfully released version $(TAG)"

#Create a git tag
git-tag:
	@echo "Creating a git tag"
	@git add .
	@git commit -m "Bumped to version $(TAG)" ;
	@git tag $(TAG) ;
	@git push --tags origin master;
	@echo 'Git tag pushed successfully' ;

#Check git status
check-git-status:
	@echo "Checking git status"
	@if [ -n "$(shell git tag | grep $(TAG))" ] ; then echo 'Tag already exists' && exit 1 ; fi
	@if [ -z "$(shell git remote -v)" ] ; then echo 'No remote to push tags to' && exit 1 ; fi
	@if [ -z "$(shell git config user.email)" ] ; then echo 'Unable to detect git credentials' && exit 1 ; fi

#Build the image
build: pre-build 
	@echo "Building docker image"
	@docker build --build-arg GOOS_VAL=$(shell go env GOOS) --build-arg GOARCH_VAL=$(shell go env GOARCH) -t $(IMAGE_REPO) -f build/Dockerfile --no-cache .
	@echo "Docker image build successfully"

#Pre-build checks
pre-build:
	@echo "Checking system information"
	@if [ -z "$(shell go env GOOS)" ] || [ -z "$(shell go env GOARCH)" ] ; then echo 'Could not determine the system architecture.' && exit 1 ; fi


#Tag images
tag-image: 
	@echo 'Tagging image'
	@docker tag $(IMAGE_REPO) $(IMAGE_REPO):$(TAG)
	@docker tag $(IMAGE_REPO) $(IMAGE_REPO):latest

#Docker push image
publish:
	@echo "Pushing docker image to repository"
	@docker login
	@docker push $(IMAGE_REPO):$(TAG)
	@docker push $(IMAGE_REPO):latest	
