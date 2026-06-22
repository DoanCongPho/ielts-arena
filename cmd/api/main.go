package main

import (
	"context"
	"github/DoanCongPho/game-arena/internal/platform"
	"github/DoanCongPho/game-arena/internal/platform/config"
	"log"
	"os"
)

func main() {
	cfg := config.MustLoad()
	_ = cfg
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		// os.Exit(database.RunMigrateCmd(cfg, migrations.FS, ocleas.Args[2:]))
	}
	plat, err := platform.Build(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = plat.Close(context.Background()) }()
}
