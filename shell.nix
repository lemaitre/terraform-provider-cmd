{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
   packages = with pkgs; [ terraform go jq ];
}

