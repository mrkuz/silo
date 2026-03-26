{
  config,
  pkgs,
  vars,
  ...
}:
{
  home = {
    username = vars.username;
    homeDirectory = "/home/${vars.username}";
    stateVersion = vars.homeStateVersion;

    shell.enableFishIntegration = true;

    packages = with pkgs; [
      age
      cloc
      colordiff
      entr
      file
      httpie
      iftop
      ncdu
      pstree
      pwgen
      rsync
      socat
      tree
      watch
      wdiff
      wget
      # C
      gcc
      pkgconf
    ];

    sessionVariables = {
      CLICOLOR = "1";
      UV_LINK_MODE = "copy";
    };
  };

  programs.home-manager.enable = true;

  programs = {
    # General
    bat.enable = true;
    eza.enable = true;
    fd.enable = true;
    fzf.enable = true;
    htop.enable = true;
    jq.enable = true;
    mise.enable = true;
    ripgrep.enable = true;
    # Python
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
        name = vars.gitUsername;
        email = vars.gitEmail;
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

  programs.opencode = {
    enable = true;
  };

  services.podman = {
    enable = true;
    # See: https://github.com/containers/image_build/blob/main/podman/Containerfile
    settings.containers = {
      containers = {
        netns = "host";
        userns = "host";
        ipcns = "host";
        utsns = "host";
        cgroupns = "host";
        cgroups = "disabled";
        log_driver = "k8s-file";
        volumes = [
          "/proc:/proc"
        ];
        default_sysctls = [ ];
      };
      engine = {
        cgroup_manager = "cgroupfs";
        events_logger = "file";
        runtime = "crun";
      };
    };
  };
}
