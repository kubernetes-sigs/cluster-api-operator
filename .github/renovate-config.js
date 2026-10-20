"use strict";

// Bot config loaded via RENOVATE_CONFIG_FILE. allowedCommands is global-only:
// Renovate ignores it if it is set in renovate.json5.
// https://docs.renovatebot.com/configuration-options/#allowedcommands
module.exports = {
  platform: "github",
  repositories: ["kubernetes-sigs/cluster-api-operator"],
  onboarding: false,
  requireConfig: "required",
  allowedCommands: ["^make modules$", "^make generate$"],
};
