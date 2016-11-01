.PHONY: dependencies
dependencies:
	@echo "Installing Glide and locked dependencies..."
	$(ECHO_V)glide --version 2>/dev/null || go get -u -f github.com/Masterminds/glide
	$(ECHO_V)glide install
	@echo "Installing test dependencies..."
	$(ECHO_V)go install ./vendor/github.com/axw/gocov/gocov
	$(ECHO_V)go install ./vendor/github.com/mattn/goveralls
	$(ECHO_V)go install ./vendor/github.com/go-playground/overalls
	@echo "Installing golint..."
	$(ECHO_V)go install ./vendor/github.com/golang/lint/golint
	@echo "Installing errcheck..."
	$(ECHO_V)go install ./vendor/github.com/kisielk/errcheck

GOCOV := gocov
OVERALLS := overalls
