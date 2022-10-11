export PATH := $(shell pwd)/bin:$(PATH)

MDOX = $(shell pwd)/../bin/mdox
MD_FILES ?= $(shell git ls-files '*.md')
MDOX_FMT_FLAGS = \
	 --soft-wraps \
	 --no-gofmt \
	 --links.validate \
	 --links.validate.config-file $(shell pwd)/.mdox-validate.yaml

.PHONY:
fmt: $(MDOX)
	$(MDOX) fmt $(MDOX_FMT_FLAGS) $(MD_FILES)

.PHONY:
check: $(MDOX)
	$(MDOX) fmt --check $(MDOX_FMT_FLAGS) $(MD_FILES)

$(MDOX):
	make -C .. $(shell pwd)/bin/mdox
