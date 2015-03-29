package main

import (
	"fmt"
	"log"
	"os"
	"ufop"
)

func help() {
	fmt.Println(`qufop v1.0

qufop [<UfopConfig>]`)
}

func main() {
	args := os.Args
	argc := len(args)

	var configFilePath string

	switch argc {
	case 2:
		configFilePath = args[1]
	default:
		help()
		return
	}

	//load config
	var ufopConf *ufop.UfopConfig
	ufopConf = &ufop.DefaultUfopConfig

	confErr := ufopConf.LoadFromFile(configFilePath)
	if confErr != nil {
		log.Println("Load config file error,", confErr)
		return
	}

	ufopServ := ufop.NewServer(ufopConf)
	ufopServ.Listen()
}
