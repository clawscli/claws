{
  description = "claws - AWS TUI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;

        indexion =
          let
            indexionHashes = {
              "aarch64-darwin" = "1w3m4px4f5hjr0776rxf3682rizz909zrvsh8i115ig67n1riqq5";
              "x86_64-linux" = "0y7dbkrxyr56kdz326hxpm6i3pljp2z4jpzv1mc5qqwrzqfgybbd";
            };
            nodePlatform = pkgs.stdenv.targetPlatform.node;
            indexionFile = "indexion-${nodePlatform.platform}-${nodePlatform.arch}.tar.gz";
          in
          pkgs.stdenvNoCC.mkDerivation (fa: {
            pname = "indexion";
            version = "0.16.0";
            src = pkgs.fetchzip {
              url = "https://github.com/trkbt10/${fa.pname}/releases/download/v${fa.version}/${indexionFile}";
              sha256 = indexionHashes.${system} or (throw "indexion: unsupported system");
              stripRoot = true;
            };
            installPhase = ''
              mkdir -p $out/bin $out/share/indexion
              cp indexion $out/bin/
              cp -r kgfs $out/share/indexion/
            '';
            meta = {
              platforms = builtins.attrNames indexionHashes;
            };
          });
      in
      {
        packages = {
          clawscli = pkgs.buildGoModule (fa: {
            pname = "claws";
            version = "git-${self.dirtyShortRev or self.shortRev}";
            # Needs to be changed whenever dependencies are updated
            vendorHash = "sha256-Ef/2Xs15E5noUYJk2J9k48g0kfTPB6v+D9uUHdOyya0=";
            src =
              let
                inherit (pkgs.lib.fileset) toSource unions;
              in
              toSource {
                root = ./.;
                fileset = unions [
                  ./go.mod
                  ./go.sum
                  ./cmd
                  ./custom
                  ./docs
                  ./internal
                  ./scripts
                  ./testdata
                ];
              };
            # Note: only building `cmd/claws` does not run the checks properly
            excludedPackages = [
              "scripts/gen-imports"
            ];
            ldflags = [
              "-s"
              "-w"
              "-X main.version=${fa.version}"
            ];
            # Replace these absolute path to echo with the nix path for testing
            postPatch = ''
              substituteInPlace internal/action/action_test.go \
                --replace-warn '/bin/echo' '${pkgs.coreutils}/bin/echo'
            '';
            # Fix home to be a writable tmp directory for config tests
            preCheck = ''
              export HOME=$(mktemp -d)
            '';
          });
          gen-imports = self.packages.${system}.clawscli.overrideAttrs (
            f: p: {
              pname = "gen-imports";
              subPackages = p.excludedPackages or [ ];
              excludedPackages = [ ];
            }
          );
          default = self.packages.${system}.clawscli;
        };
        devShells.default = pkgs.mkShell {
          packages =
            with pkgs;
            [
              go_1_25
              go-task
              gopls
              golangci-lint
              vhs
              ttyd
              nodejs
              bash
            ]
            ++ lib.optionals (builtins.elem pkgs.stdenv.targetPlatform.system indexion.meta.platforms) [
              indexion
            ];

          env.GOROOT = "${pkgs.go_1_25}/share/go";

          shellHook = ''
            echo "claws dev env - Go $(go version | cut -d' ' -f3)"
          '';
        };
      }
    );
}
