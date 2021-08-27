{ lib, buildGoModule, fetchFromGitHub, makeWrapper}:

buildGoModule rec {
  pname = "tilt";
  /* Do not use "dev" as a version. If you do, Tilt will consider itself
    running in development environment and try to serve assets from the
    source tree, which is not there once build completes.  */
  version = "0.22.5";

  src = fetchFromGitHub {
    owner = "tilt-dev";
    repo = pname;
    rev = "39122ff70"; # right after v0.22.5
    sha256 = null;
  };
  vendorSha256 = null;

  subPackages = [ "cmd/tilt" ];

  buildInputs = [makeWrapper];

  meta = with lib; {
    description = "Local development tool to manage your developer instance when your team deploys to Kubernetes in production";
    homepage = "https://tilt.dev/";
    license = licenses.asl20;
    maintainers = with maintainers; [ anton-dessiatov ];
  };
  doCheck = false;
  # Explicitly ask to use the upstream-hosted web assets
  postFixup = "wrapProgram $out/bin/tilt --set TILT_WEB_MODE prod";
}
