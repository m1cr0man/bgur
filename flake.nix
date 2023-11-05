{
  description = "BGur - Imgur desktop backgrounds";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        devDependencies = with pkgs; [
          cmake
          go
          gopls
          go-outline
          gopkgs
          # These paths dont actually work for the vscode extension but kept anyway
          godef
          delve
          gotools
        ];
      in
      rec {
        devShells.default = pkgs.mkShell {
          buildInputs = devDependencies;
        };

        devShell = devShells.default;
      }
    );
}
