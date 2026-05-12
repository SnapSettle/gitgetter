{
  description = "gitgetter — a modern TUI for everyday git operations";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, flake-utils, treefmt-nix }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = import nixpkgs { inherit system; };

          # ── treefmt ────────────────────────────────────────────────────────
          treefmtEval = treefmt-nix.lib.evalModule pkgs {
            projectRootFile = "flake.nix";
            programs = {
              gofmt.enable = true; # Go sources
              nixpkgs-fmt.enable = true; # Nix files
              prettier.enable = true; # Markdown / YAML / JSON
            };
          };

          # ── package ────────────────────────────────────────────────────────
          gitgetter = pkgs.buildGoModule rec {
            pname = "gitgetter";
            version = "1.0.0";
            src = self;

            vendorHash = "sha256-FA/ttb1dwveAmEIrgOHepP349Q+ST4V0AR3dE8Ku7iE=";

            meta = with pkgs.lib; {
              description = "A modern TUI for everyday git operations";
              homepage = "https://github.com/snapsettle/gitgetter";
              license = licenses.mit;
              maintainers = [ ];
              mainProgram = "gitgetter";
            };
          };


        in
        {
          # `nix build`
          packages.default = gitgetter;
          packages.gitgetter = gitgetter;

          # `nix run`
          apps.default = flake-utils.lib.mkApp { drv = gitgetter; };

          # `nix fmt`
          formatter = treefmtEval.config.build.wrapper;

          # `nix flake check` includes format check
          checks = {
            formatting = treefmtEval.config.build.check self;
          };

          # `nix develop`
          devShells.default = pkgs.mkShell {
            name = "gitgetter-dev";

            packages = with pkgs; [
              # Go toolchain
              go
              gopls
              golangci-lint
              delve # debugger

              # Git (runtime dep)
              git

              # Nix tooling
              nil # Nix LSP
              nixpkgs-fmt

              # Formatting
              treefmtEval.config.build.wrapper
            ];

            shellHook = ''
              echo ""
              echo "  ╭───────────────────────────────────────╮"
              echo "  │   gitgetter dev shell  🐙              │"
              echo "  │                                       │"
              echo "  │   go build  ./...   – build           │"
              echo "  │   go run    .       – run             │"
              echo "  │   go test   ./...   – test            │"
              echo "  │   nix fmt           – format all      │"
              echo "  ╰───────────────────────────────────────╯"
              echo ""
            '';
          };
        }
      ) // {
      # NixOS / nix-darwin module (see nix/module.nix)
      nixosModules.default = import ./nix/module.nix { inherit self; };
      darwinModules.default = import ./nix/module.nix { inherit self; };
    };
}
