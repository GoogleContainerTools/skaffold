{
  description = "Easy and Repeatable Kubernetes Development";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts.url = "github:hercules-ci/flake-parts";

    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {

      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];

      perSystem = { pkgs, inputs', ... }: {

        packages.default = let
          buildDate = with inputs; "${self.lastModifiedDate or self.lastModified or "unknown"}";
          version  = with inputs; "${self.shortRev or self.dirtyShortRev or buildDate}";
        in inputs'.gomod2nix.legacyPackages.buildGoApplication {
          pname = "skaffold";

          inherit version;

          src = inputs.self;

          modules = null;

          subPackages = ["cmd/skaffold"];

          ldflags = let
            p = "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold";
          in [
            "-s" "-w"
            "-X ${p}/version.version=v${version}"
            "-X ${p}/version.gitCommit=${inputs.self.rev or inputs.self.dirtyRev or "unknown"}"
            "-X ${p}/version.buildDate=${buildDate}"
          ];

          nativeBuildInputs = with pkgs; [ installShellFiles makeWrapper ];

          installCheckPhase = ''
            $out/bin/skaffold version | grep ${version} > /dev/null
          '';

          postInstall = ''
            wrapProgram $out/bin/skaffold --set SKAFFOLD_UPDATE_CHECK false

            installShellCompletion --cmd skaffold \
              --bash <($out/bin/skaffold completion bash) \
              --zsh <($out/bin/skaffold completion zsh)
          '';

          meta = {
            homepage = "https://github.com/GoogleContainerTools/skaffold";
          };
        };

        packages.gomod2nix = inputs'.gomod2nix.packages.default;
      };
    };
}
