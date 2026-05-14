{ config, pkgs, lib, ... }:
let
  vars = {
    gitUserName = "mrkuz";
    gitUserEmail = "master23@gmail.com";
  };
in
{
  silo.shellCommand = "${pkgs.fish}/bin/fish --login";

  home = {
    shell.enableFishIntegration = true;

    packages = with pkgs; [
      age
      cloc
      colordiff
      entr
      file
      httpie
      iftop
      nano
      ncdu
      procps
      pstree
      pwgen
      python3
      rsync
      socat
      tree
      watch
      wdiff
      wget
      which
      # C
      gcc
      pkgconf
    ];

    sessionVariables = {
      CLICOLOR = "1";
      UV_LINK_MODE = "copy";
    };

    sessionPath = [
      "$HOME/.local/bin"
      "$HOME/.npm/bin"
      "$HOME/bin"
      "$HOME/go/bin"
    ];
  };

  programs = {
    bat.enable = true;
    fd.enable = true;
    fzf.enable = true;
    htop.enable = true;
    jq.enable = true;
    less.enable = true;
    npm.enable = true;
    ripgrep.enable = true;
    uv.enable = true;
  };

  programs.diff-so-fancy = {
    enable = true;
    enableGitIntegration = true;
  };

  programs.fish = {
    enable = true;
    plugins = [
      {
        name = "pure";
        src = pkgs.fishPlugins.pure.src;
      }
      {
        name = "async-prompt";
        src = pkgs.fishPlugins.async-prompt.src;
      }
    ];
    shellAbbrs = {
      gau = "git add -u";
      gc = "git commit";
      gcm = "git commit -m";
      gcmm = "git checkout --";
      gd = "git diff";
      gdc = "git diff --cached";
      gs = "git status";
    };
    interactiveShellInit = ''
      set -U fish_greeting
      set -U pure_symbol_prompt ">>"
      set -U pure_color_mute "brgreen"
      set -U pure_enable_nixdevshell true
      set -U pure_enable_single_line_prompt false
      set -U fish_color_autosuggestion 586e75
      fish_add_path $HOME/bin
      fish_add_path $HOME/.local/bin/
    '';
  };

  programs.git = {
    enable = true;
    settings = {
      user = {
        name = vars.gitUserName;
        email = vars.gitUserEmail;
      };
      init = {
        defaultBranch = "main";
      };
      merge = {
        ff = false;
      };
      pull = {
        rebase = true;
      };
    };
  };

  programs.mise = {
    enable = true;
    globalConfig = {
      tools = {
        claude-code = "latest";
        opencode = "latest";
      };
    };
  };

  home.activation = {
    mise = lib.hm.dag.entryAfter ["writeBoundary"] ''
      ${pkgs.mise}/bin/mise install
    '';
    npm = lib.hm.dag.entryAfter ["writeBoundary"] ''
      export PATH=''$PATH:${pkgs.nodejs}/bin/
      ${pkgs.nodejs}/bin/npm install -g @fission-ai/openspec@latest
    '';
  };
}
