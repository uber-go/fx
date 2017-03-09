.PHONY: libdeps
libdeps:
	@$(call label,Installing Glide and locked dependencies...)
	$(ECHO_V)glide --version ||	go get -v github.com/Masterminds/glide
	$(ECHO_V)glide install

.PHONY: deps
deps: libdeps
	@$(call label,Installing test dependencies...)
	$(ECHO_V)go install ./vendor/github.com/axw/gocov/gocov
	$(ECHO_V)go install ./vendor/github.com/mattn/goveralls
	$(ECHO_V)go install ./vendor/github.com/go-playground/overalls
	@$(call label,Installing golint...)
	$(ECHO_V)go install ./vendor/github.com/golang/lint/golint
	@$(call label,Installing errcheck...)
	$(ECHO_V)go install ./vendor/github.com/kisielk/errcheck
	@$(call label,Installing md-to-godoc...)
	$(ECHO_V)go install ./vendor/github.com/sectioneight/md-to-godoc
	@$(call label,Installing interfacer...)
	$(ECHO_V)go install ./vendor/github.com/mvdan/interfacer/cmd/interfacer
	@$(call label,Installing richgo...)
	$(ECHO_V)go install ./vendor/github.com/kyoh86/richgo

GOCOV := gocov
OVERALLS := overalls
