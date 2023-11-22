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

export PATH := $(abspath $(TOOLS_BIN_DIR)/bin):$(PATH)

# Only set MDBOOK if it's not set as an environment variable
MDBOOK ?= $(TOOLS_BIN_DIR)/bin/mdbook
$(MDBOOK):
	$(MDBOOK_INSTALL) 0.4.35 $(TOOLS_BIN_DIR)

.PHONY: serve
serve: $(MDBOOK)
	$(MDBOOK) serve

.PHONY: build
build: $(MDBOOK)
	$(MDBOOK) build
	cp $(ROOT_DIR)/index.yaml book
