package common

import (
	"log"

	"github.com/Ghytro/ab_interview/config"
)

func LogIfVerbose(message interface{}) {
	if config.Config.IsVerbose {
		log.Println(message)
	}
}
