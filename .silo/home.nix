{
  config,
  pkgs,
  ...
}:
{
  module.podman.enable = false;
  programs.go.enable = true;
  home.packages = with pkgs; [
    golangci-lint
    gopls
    go-tools
  ];
}
