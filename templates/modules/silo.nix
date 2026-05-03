{ config, pkgs, lib, ... }:
{
  options.silo = {
    shellCommand = lib.mkOption {
      type = lib.types.str;
      default = "${pkgs.bash}/bin/bash --login";
      description = "Default silo shell command";
    };
  };

  config = {
    home.packages = [
      (pkgs.writeShellScriptBin "shell" "exec ${config.silo.shellCommand}")
    ];
  };
}
