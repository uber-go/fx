export PATH := $(shell pwd)/bin:$(PATH)

MDOX = $(shell pwd)/../bin/mdox
MDOX_FMT_FLAGS = --soft-wraps --links.validate
MD_FILES = $(shell find . -name '*.md')

.PHONY:
fmt: $(MDOX)
	@$(foreach FILE,$(MD_FILES), \
		(cd $(dir $(FILE)) && \
		echo "[mdox fmt] $(FILE)" && \
		$(MDOX) fmt $(MDOX_FMT_FLAGS) $(notdir $(FILE)));)

.PHONY:
check: $(MDOX)
	@$(foreach FILE,$(MD_FILES), \
		(cd $(dir $(FILE)) && \
		echo "[mdox check] $(FILE)" && \
		$(MDOX) fmt --check $(MDOX_FMT_FLAGS) $(notdir $(FILE)));)

$(MDOX):
	make -C .. $(shell pwd)/bin/mdox
