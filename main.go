package main

import (
	"log"

	"github.com/ability-sh/abi-ac-driver/driver"
	"github.com/ability-sh/abi-app-store/srv"
	_ "github.com/ability-sh/abi-db/client/service"
	_ "github.com/ability-sh/abi-micro/http"
	_ "github.com/ability-sh/abi-micro/logger"
	_ "github.com/ability-sh/abi-micro/redis"
	_ "github.com/ability-sh/abi-micro/smtp"
)

func main() {
	err := driver.Run(driver.NewReflectExecutor(&srv.Server{}))
	if err != nil {
		log.Fatalln(err)
	}
}
