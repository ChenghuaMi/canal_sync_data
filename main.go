package main

import (
	"fmt"
	"log"
	"net/http"
	"orca_trade_data/pkg"
	"orca_trade_data/pkg/client_canal"
)

func main() {
	cfg := pkg.Viper()
	log.Printf("listen:%s", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))
	canal := client_canal.NewClient(cfg)

	go func() {
		if err := canal.HandleData(); err != nil {

		}
	}()
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port), nil); err != nil {
		log.Fatalf("server run error: %v", err)
	}
}
