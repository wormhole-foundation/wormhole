SUBDIRS := $(wildcard */)

TARGETS := build test

.PHONY: $(TARGETS)
$(TARGETS):
	$(foreach dir,$(SUBDIRS),make -C $(dir) $@ &&) true
