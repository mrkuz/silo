{
  config,
  pkgs,
  ...
}:
{
  programs.go.enable = true;
  home.packages = with pkgs; [
    golangci-lint
    gopls
    go-tools
  ];
}
