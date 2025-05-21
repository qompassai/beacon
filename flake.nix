# ~/.GH/Qompass/Go/beacon/flake.nix
# ---------------------------------
# Copyright (C) 2025 Qompass AI, All rights reserved

{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";

  outputs = { self, nixpkgs }: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    devShells.${system}.default = pkgs.mkShell {
      packages = [ 
        pkgs.go_1_24
        pkgs.zig
      ];

      shellHook = ''
        echo "Using Go $(go version)"
        echo "Using Zig $(zig version)"
      '';
    };
  };
}
