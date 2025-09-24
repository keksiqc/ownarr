{pkgs}: {
  channel = "unstable";
  packages = [
    pkgs.go
    pkgs.golangci-lint
    pkgs.gnumake
    pkgs.qwen-code
  ];
  services = {
    docker.enable = true;
  };
  idx.extensions = [
    "golang.go"
    "ms-azuretools.vscode-docker"
    "EditorConfig.EditorConfig"
    "usernamehw.errorlens"
    "tamasfe.even-better-toml"
    "Codeium.codeium"
    "antfu.icons-carbon"
    "antfu.file-nesting"
    "redhat.vscode-yaml"
    "ms-azuretools.vscode-containers"
  ];
}