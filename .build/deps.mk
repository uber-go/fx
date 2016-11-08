.PHONY: dependencies
dependencies:
	@$(call label,Installing Glide and locked dependencies...)
	$(ECHO_V)glide --version 2>/dev/null || go get -u -f github.com/Masterminds/glide
	$(ECHO_V)glide install
	@$(call label,Installing test dependencies...)
	$(ECHO_V)go install ./vendor/github.com/axw/gocov/gocov
	$(ECHO_V)go install ./vendor/github.com/mattn/goveralls
	$(ECHO_V)go install ./vendor/github.com/go-playground/overalls
	@$(call label,Installing golint...)
	$(ECHO_V)go install ./vendor/github.com/golang/lint/golint
	@$(call label,Installing errcheck...)
	$(ECHO_V)go install ./vendor/github.com/kisielk/errcheck

GOCOV := gocov
OVERALLS := overalls
