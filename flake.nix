{
  description = "Gala - Git Author Line Analyzer";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      goVersion = 24; # Go 1.24
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      # Version and build info
      version = "1.0.0";
      gitCommit = self.rev or self.dirtyRev or "dev";

    in
    flake-utils.lib.eachSystem supportedSystems (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ self.overlays.default ];
        };

        # Build Gala package
        gala = pkgs.buildGoModule rec {
          pname = "gala";
          version = "1.0.0";

          src = ./.;

          # Update this when go.mod/go.sum changes
          vendorHash = "sha256-SD5gL01LBrcUUhp2Z9SUqc10vSZjCrwx8KucGQFJsaU=";

          # Build flags
          ldflags = [
            "-s"
            "-w"
            "-X main.Version=${version}"
            "-X main.GitCommit=${gitCommit}"
          ];

          # Build inputs
          nativeBuildInputs = with pkgs; [
            git # Required for git operations
            makeWrapper # Required for wrapProgram
          ];

          # Runtime dependencies
          buildInputs = with pkgs; [
            git
          ];

          # Ensure git is available at runtime
          postInstall = ''
            wrapProgram $out/bin/gala \
              --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.git ]}
          '';

          # Metadata
          meta = with pkgs.lib; {
            description = "A high-performance command-line tool for analyzing git repository contributions by counting lines authored by different contributors.";
            homepage = "https://github.com/doprz/gala";
            license = licenses.mit;
            maintainers = with maintainers; [
              "doprz"
            ];
            platforms = platforms.unix;
            mainProgram = "gala";
          };
        };

      in
      {
        # Main package
        packages = {
          default = gala;
          gala = gala;
        };

        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go
            gotools
            golangci-lint

            # Git for development and testing
            git

            # Build tools
            gnumake

            # Optional: useful development tools
            # jq # For JSON processing
            # yq # For YAML processing
          ];

          # Development environment
          shellHook = ''
            echo "Gala development environment"
            echo "Go version: $(go version)"
            echo "Git version: $(git --version)"
            echo ""
          '';

          # Environment variables for development
          GALA_DEV = "1";
          CGO_ENABLED = "0";
        };

        # Applications for `nix run`
        apps = {
          default = {
            type = "app";
            program = "${gala}/bin/gala";
          };

          gala = {
            type = "app";
            program = "${gala}/bin/gala";
          };
        };

        # Formatter for `nix fmt`
        formatter = pkgs.nixfmt-rfc-style;
      }
    )
    // {
      # Global overlay
      overlays.default = final: prev: {
        go = final."go_1_${toString goVersion}";
        gala = self.packages.${final.system}.gala;
      };

      # NixOS module
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        with lib;
        let
          cfg = config.services.gala;
        in
        {
          options.services.gala = {
            enable = mkEnableOption "Gala git analyzer service";

            package = mkOption {
              type = types.package;
              default = self.packages.${pkgs.system}.gala;
              description = "The Gala package to use";
            };
          };

          config = mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];
          };
        };

      # Home Manager module
      homeManagerModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        with lib;
        let
          cfg = config.programs.gala;
        in
        {
          options.programs.gala = {
            enable = mkEnableOption "Gala git analyzer";

            package = mkOption {
              type = types.package;
              default = self.packages.${pkgs.system}.gala;
              description = "The Gala package to use";
            };

            settings = mkOption {
              type = types.attrsOf types.anything;
              default = { };
              description = "Configuration for Gala";
              example = {
                output = "table";
                emoji = true;
                concurrency = 8;
              };
            };
          };

          config = mkIf cfg.enable {
            home.packages = [ cfg.package ];

            # Create config file if settings are provided
            home.file.".config/gala/gala.yaml" = mkIf (cfg.settings != { }) {
              text = builtins.toJSON cfg.settings;
            };
          };
        };

      # Hydra jobsets (for CI)
      hydraJobs = self.packages;
    };
}
