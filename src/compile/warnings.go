package main

import "github.com/cloudfoundry/libbuildpack"

func WarnNodeEngine(nodeEngine string, logger libbuildpack.Logger) {
	if nodeEngine == "" {
		logger.Protip("Node version not specified in package.json", "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html")
	} else if nodeEngine == "*" {
		logger.Protip("Dangerous semver range (*) in engines.node", "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html")
	} else if nodeEngine[0] == '>' {
		logger.Protip("Dangerous semver range (>) in engines.node", "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html")
	}
}
