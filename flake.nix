{
  description = "claws - AWS TUI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_25
            go-task
            golangci-lint
          ];

          shellHook = ''
            echo "claws dev env - Go $(go version | cut -d' ' -f3)"
          '';
        };
      }
    );
}
