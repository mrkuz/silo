{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      nixpkgs,
      home-manager,
      ...
    }:
    let
      vars = {
        username = "markus";
        gitUsername = "mrkuz";
        gitEmail = "markus@bitsandbobs.net";
        system = "aarch64-linux";
        homeStateVersion = "25.11";
      };
    in
    {
      homeConfigurations."${vars.username}" = home-manager.lib.homeManagerConfiguration {
        pkgs = import nixpkgs {
          system = vars.system;
          config.allowUnfree = true;
        };

        extraSpecialArgs = {
          inherit vars;
        };

        modules = [
          ./home.nix
        ];
      };
    };
}
