{
  description = "MCAP to MATLAB Converter";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs = { self, nixpkgs }: 
    let 
      project_overlay = final: prev: {
        go_application = final.callPackage ./mcap_reader.nix { };
        python_application = final.callPackage ./matlab_generator.nix { };
      };
      my_overlays = [ project_overlay ];
      pkgs = import nixpkgs {
        system = "x86_64-linux";
        overlays = [ self.overlays.default ];
      };
    in 
    {

      overlays.default = nixpkgs.lib.composeManyExtensions my_overlays;
      packages.x86_64-linux =
        rec {
          go_application = pkgs.go_application;
          python_application = pkgs.python_application;
          default = go_application;
        };
      
        devShells.x86_64-linux.default =
          pkgs.mkShell rec {
            name = "nix-devshell";
            inputsFrom = [ pkgs.python_application ];
            packages = with pkgs; [
              # Development Tools
              mcap-cli
              go_application
              python_application
              python311Packages.scipy
            ];

            # Setting up the environment variables you need during
            # development.
            shellHook =
              let
                icon = "f121";
              in
              ''
                export PS1="$(echo -e '\u${icon}') {\[$(tput sgr0)\]\[\033[38;5;228m\]\w\[$(tput sgr0)\]\[\033[38;5;15m\]} (${name}) \\$ \[$(tput sgr0)\]"
              '';
        }; 
    };
}
