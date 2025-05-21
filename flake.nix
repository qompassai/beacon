# ~/.GH/Qompass/Go/beacon/flake.nix
# ---------------------------------
# Copyright (C) 2025 Qompass AI, All rights reserved

{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";

  outputs = { self, nixpkgs }: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    # Dev shell configuration
    devShells.${system}.default = pkgs.mkShell {
      packages = [ pkgs.go pkgs.zig ];
    };

    # Add package output
    packages.${system}.default = pkgs.stdenv.mkDerivation {
      name = "beacon";
      src = ./.;
      
      buildInputs = [ pkgs.go pkgs.zig ];
      
      buildPhase = ''
        go build -o beacon
      '';
      
      installPhase = ''
        mkdir -p $out/bin
        cp beacon $out/bin/
      '';
    };
  };
}
