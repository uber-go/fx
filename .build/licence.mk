LICENCE_FORMAT := "*.go"

LICENCE_BIN := uber-licence

add-uber-licence:
	which $(LICENCE_BIN) > /dev/null || npm install -g $(LICENCE_BIN)
	$(LICENCE_BIN) --file $(LICENCE_FORMAT)
