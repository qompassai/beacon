# ~/.GH/Qompass/Go/beacon/default.nix
# -----------------------------------
# Copyright (C) 2025 Qompass AI, All rights reserved

{ pkgs ? import <nixpkgs> {} }:

let
  zigOverlay = final: prev: {
    zig-master = prev.stdenv.mkDerivation rec {
      name = "zig";
      version = "0.15.0";  # Updated version

      src = prev.fetchFromGitHub {
        owner = "ziglang";
        repo = "zig";
        rev = "d7802f2c62a0e8a8bf7aae2427750d8a25e1c7c1";
        hash = "sha256-5kDdMNZ6hqH2ESyB7k3qFhG0l6i1jQmYtZJv8W4X3qY=";
      };

      nativeBuildInputs = [ prev.cmake ];
      buildInputs = [ prev.llvmPackages_18.llvm ];
    };
  };

  pkgs = import <nixpkgs> {
    overlays = [ zigOverlay ];
  };

in pkgs.stdenv.mkDerivation {
 name = "beacon";
  src = ./.;

  buildInputs = [
    pkgs.go
    pkgs.zig-master
  ];

  buildPhase = ''
    export GOPATH="$src"
    export CGO_ENABLED=1
    export CC="zig cc"
    export CXX="zig c++"
    
    go build -trimpath -o beacon
    
    zig build -Doptimize=ReleaseSafe
  '';

  installPhase = ''
    mkdir -p $out/bin
    cp beacon $out/bin/
    cp zig-out/bin/* $out/bin/
  '';
}


