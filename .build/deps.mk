.PHONY: libdeps
libdeps:
	@$(call label,Installing Glide and locked dependencies...)
	$(ECHO_V)glide --version 2>/dev/null ||	go get -v github.com/Masterminds/glide
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
	$(ECHO_V)echo "--- PASS: TestSomething" | richgo testfilter > /dev/null 2>&1 || ($(call label,Installing richgo) && go get github.com/sectioneight/richgo)

GOCOV := gocov
OVERALLS := overalls
