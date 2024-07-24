package tf

import (
	"github.com/nttcom/ksot/nb-server/pkg/model"
)

var TfLogic = model.NewPathMapLogic()

func init() {
	// User needs to create a function to generate a pathmap and add it to the MAP.
	// Example.
	// TfLogic["serviceName"] = serviceName
}
