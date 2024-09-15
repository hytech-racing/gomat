{ lib, buildGoModule, makeWrapper, fetchFromGitHub }:

buildGoModule rec {
  pname = "mcap_reader";
  version = "1.0.0";
  src = ./mcap_reader;
  vendorHash = "sha256-P8uEvkngir0xjgSgKC9et6lWz00rv6Pi3JIK9JAN0Rc=";

  meta = with lib; {
    description = "Reads and parses the mcap data";
    license = licenses.mit;
    platforms = platforms.linux;
  };
}
