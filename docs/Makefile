export PATH := $(shell pwd)/bin:$(PATH)

MDOX = $(shell pwd)/../bin/mdox
MDOX_FMT_FLAGS = --soft-wraps --links.validate --links.validate.config-file $(shell pwd)/.mdox-validate.yaml
MD_FILES ?= $(shell git ls-files '*.md')

.PHONY:
fmt: $(MDOX)
	$(MDOX) fmt $(MDOX_FMT_FLAGS) $(MD_FILES)

.PHONY:
check: $(MDOX)
	$(MDOX) fmt --check $(MDOX_FMT_FLAGS) $(MD_FILES)

$(MDOX):
	make -C .. $(shell pwd)/bin/mdox
