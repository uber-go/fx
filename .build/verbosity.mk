V ?= 0
ifeq ($(V),0)
  ECHO_V = @
else
	TEST_VERBOSITY_FLAG = -v
	DEBUG_FLAG = -debug
endif
