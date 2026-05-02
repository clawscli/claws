{
  description = "claws - AWS TUI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;

        indexionPlatform = {
          "aarch64-darwin" = "darwin-arm64";
          "x86_64-linux" = "linux-x64";
        }.${system} or (throw "indexion: unsupported system ${system}");

        indexionHash = {
          "darwin-arm64" = "1gbjfzwy9rgn7n79hj354w1jh2cqc6fvsj1m2zscvg6va6b1hdhl";
          "linux-x64" = "1pqll5vkb50fygq7ibqdry0lby54r50p17f75fv2s95xqy515c3i";
        }.${indexionPlatform};

        indexion = pkgs.stdenvNoCC.mkDerivation {
          pname = "indexion";
          version = "0.11.0";
          src = pkgs.fetchzip {
            url = "https://github.com/trkbt10/indexion/releases/download/v0.11.0/indexion-${indexionPlatform}.tar.gz";
            sha256 = indexionHash;
            stripRoot = true;
          };
          installPhase = ''
            mkdir -p $out/bin $out/share/indexion
            cp indexion $out/bin/
            cp -r kgfs $out/share/indexion/
          '';
        };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_25
            go-task
            gopls
            golangci-lint
            vhs
            ttyd
            indexion
            nodejs
            bash
          ];

          env.GOROOT = "${pkgs.go_1_25}/share/go";

          shellHook = ''
            echo "claws dev env - Go $(go version | cut -d' ' -f3)"
          '';
        };
      }
    );
}
