package main

import (
	"log"

	"github.com/ability-sh/abi-ac-driver/driver"
	"github.com/ability-sh/abi-app-store/srv"
	_ "github.com/ability-sh/abi-micro/grpc"
	_ "github.com/ability-sh/abi-micro/logger"
	_ "github.com/ability-sh/abi-micro/mongodb"
	_ "github.com/ability-sh/abi-micro/oss"
	_ "github.com/ability-sh/abi-micro/redis"
	_ "github.com/ability-sh/abi-micro/smtp"
)

func main() {
	err := driver.Run(driver.NewReflectExecutor(&srv.Server{}))
	if err != nil {
		log.Fatalln(err)
	}
}
