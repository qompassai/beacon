# ~/.GH/Qompass/Go/beacon/shell.nix
# ---------------------------------
# Copyright (C) 2025 Qompass AI, All rights reserved

{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    (import ./default.nix { inherit pkgs; }).buildInputs
    pkgs.gopls
    pkgs.zls
  ];

  shellHook = ''
    echo "Go version: $(go version)"
    echo "Zig version: $(zig version)"
    export CC="zig cc"
    export CXX="zig c++"
  '';
}

