{ pkgs, ... }:

{

  languages = {
    zig.enable = true;
  };

  services.mysql = {
    enable = true;
    package = pkgs.mysql80;
    initialDatabases = [
      { name = "mydb"; }
    ];
    ensureUsers = [
      {
        name = "devuser";
        password = "devpass";
        ensurePermissions = {
          "mydb.*" = "ALL PRIVILEGES";
        };
      }
    ];
  };

}
