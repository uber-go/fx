_THIS_LICENCE_MAKEFILE := $(lastword $(MAKEFILE_LIST))
_THIS_LICENCE_DIR := $(dir $(_THIS_LICENCE_MAKEFILE))

add-uber-licence:
	$(ECHO_V)OUTPUT=$($(_THIS_LICENCE_DIR)/check_licence.sh) ; \
		[ -z "$$(OUTPUT)" ] || (echo $$(OUTPUT) && exit 1)
