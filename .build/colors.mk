# Colorization rules
_NCOLORS=$(shell test -n $(TERM) && tput colors 2> /dev/null || echo 0)
TERMINAL_HAS_COLORS ?= 0

ifeq ($(shell test -n "$(_NCOLORS)" -a "$(_NCOLORS)" -ge 8 && echo y),y)
  COLOR_BOLD = "$(shell tput bold)"
  COLOR_NORMAL = "$(shell tput sgr0)"
  COLOR_COMMAND = "$(shell tput setaf 6)"
  COLOR_ERROR = "$(shell tput setaf 1)"
  COLOR_RESET = "$(COLOR_NORMAL)"
  TERMINAL_HAS_COLORS = 1
endif

LABEL_STYLE = "$(COLOR_BOLD)$(COLOR_COMMAND)"
ERROR_STYLE = "$(COLOR_BOLD)$(COLOR_ERROR)"

label = echo $(COLOR_BOLD)$(COLOR_COMMAND)$(1)$(COLOR_NORMAL)
die = (echo $(COLOR_BOLD)$(COLOR_ERROR)$(1)$(COLOR_NORMAL); exit 1)
