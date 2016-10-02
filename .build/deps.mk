.PHONY: dependencies
dependencies:
	@echo "Installing Glide and locked dependencies..."
	$(ECHO_V)glide --version || go get -u -f github.com/Masterminds/glide
	$(ECHO_V)glide install
	@echo "Installing test dependencies..."
	$(ECHO_V)go install ./vendor/github.com/axw/gocov/gocov
	$(ECHO_V)go install ./vendor/github.com/mattn/goveralls
	$(ECHO_V)go install ./vendor/github.com/go-playground/overalls
ifdef SHOULD_LINT
	@echo "Installing golint..."
	$(ECHO_V)go install ./vendor/github.com/golang/lint/golint
else
	@echo "Not installing golint, since we don't expect to lint on" $(GO_VERSION)
endif

GOCOV := gocov
OVERALLS := overalls
