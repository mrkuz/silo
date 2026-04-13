{ config, pkgs, lib, ... }:
let
  cfg = config.module.podman;
in
{
  options.module.podman = {
    enable = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Enable Podman service for nested containers";
    };
  };

  config = lib.mkIf cfg.enable {
    services.podman = {
      enable = true;
      settings.containers = {
        containers = {
          netns = "host";
          userns = "host";
          ipcns = "host";
          utsns = "host";
          cgroupns = "host";
          cgroups = "disabled";
          log_driver = "k8s-file";
          volumes = [ "/proc:/proc" ];
          default_sysctls = [ ];
        };
        engine = {
          cgroup_manager = "cgroupfs";
          events_logger = "file";
          runtime = "crun";
        };
      };
    };
  };
}
