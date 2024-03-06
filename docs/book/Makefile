# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Directories.
ROOT_DIR := $(realpath ../..)
TOOLS_DIR := $(realpath ../../hack/tools)
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
BIN_DIR := bin
MDBOOK_INSTALL := $(realpath ../../scripts/ci-install-mdbook.sh)
TABULATE := $(TOOLS_BIN_DIR)/mdbook-tabulate
EMBED := $(TOOLS_BIN_DIR)/mdbook-embed
RELEASELINK := $(TOOLS_BIN_DIR)/mdbook-releaselink
MDBOOK := $(TOOLS_BIN_DIR)/bin/mdbook
FS_SUMMARY := $(TOOLS_BIN_DIR)/bin/mdbook-fs-summary

export PATH := $(abspath $(TOOLS_BIN_DIR)/bin):$(PATH)

BOOK_DEPS := $(MDBOOK) $(TABULATE) $(EMBED) $(RELEASELINK) $(FS_SUMMARY)

$(TOOLS_BIN_DIR)/%: $(TOOLS_DIR_DEPS)
	make -C $(TOOLS_DIR) $(subst $(TOOLS_DIR)/,,$@)

$(MDBOOK):
	$(MDBOOK_INSTALL) 0.4.37 $(TOOLS_BIN_DIR)

.PHONY: serve
serve: $(BOOK_DEPS) ## Run a local web server with the compiled book
	$(MDBOOK) serve

.PHONY: build
build: $(BOOK_DEPS) ## Build the book
	$(MDBOOK) build
	cp $(ROOT_DIR)/index.yaml book

.PHONY: clean
clean:
	rm -rf book