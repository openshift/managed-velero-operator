# This must come before the boilerplate includes, which otherwise
# default the version.
VERSION_MINOR?=2

include boilerplate/generated-includes.mk

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
