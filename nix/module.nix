{ self }:
{ config, lib, pkgs, ... }:

let
  cfg = config.programs.gitgetter;
in
{
  # ── Option declarations ────────────────────────────────────────────────────
  options.programs.gitgetter = {
    enable = lib.mkEnableOption "gitgetter — a modern TUI for everyday git operations";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "gitgetter.packages.\${system}.default";
      description = "The gitgetter package to install.";
    };

    # Optional: put gitgetter in the system PATH and add a shell alias.
    alias = lib.mkOption {
      type = lib.types.str;
      default = "gg";
      description = ''
        Shell alias for gitgetter.  Set to an empty string to disable.
      '';
    };
  };

  # ── Implementation ────────────────────────────────────────────────────────
  config = lib.mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];

    # Add an optional short alias to every user's shell init.
    environment.shellAliases = lib.mkIf (cfg.alias != "") {
      ${cfg.alias} = "gitgetter";
    };
  };
}
