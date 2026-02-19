{ pkgs, config, ... }:

{

  languages = {
    go.enable = true;
  };

  packages = with pkgs; [
    sqlite
    goose
  ];

  env = {
    DEV_DB_PATH = "${config.git.root}/dev.db";
    DEV_LOG_PATH = "${config.git.root}/dev.log";
  };

  scripts = {
    run.exec = ''
      go run . --db "$DEV_DB_PATH" --log "$DEV_LOG_PATH"
    '';

    sqlite.exec = ''sqlite3 "$DEV_DB_PATH"'';

    migrateCreate.exec = ''goose -dir ./migrations create "$1" sql'';
    migrateUp.exec = ''goose -dir ./migrations sqlite3 "$DEV_DB_PATH" up'';
    migrateStatus.exec = ''goose -dir ./migrations sqlite3 "$DEV_DB_PATH" status'';
  };

}
