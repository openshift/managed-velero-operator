# This must come before the boilerplate includes, which otherwise
# default the version.
VERSION_MINOR?=2

include boilerplate/generated-includes.mk

# >> TEMPORARY >>
# Remove this section once prow configuration is standardized.
.PHONY: verify
verify: lint
# << TEMPORARY <<

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
