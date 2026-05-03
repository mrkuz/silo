{
  config,
  pkgs,
  ...
}:
{
  silo.podman.enable = false;
  programs.go.enable = true;
  home.packages = with pkgs; [
    delve
    golangci-lint
    gopls
    go-tools
  ];
}
