package builder

import (
	"flag"
)

var (
	NO_MANIFEST_EXISTS="Manifest file does not exist"
)

type Builder struct {
	Config Config
}

type Config struct {
	Manifest string
	Ami  	 string
	Env  	 string
	Confirm  bool
}

func New() Builder {
	config := parserConfigFile()


	// Get New Builder
	builder := Builder{Config: config}

	return builder
}

// Parsing Config from command
func parserConfigFile() Config {
	manifest := flag.String("manifest", "", "The manifest configuration file to use.")
	ami := flag.String("ami", "", "The AMI to use for the servers.")
	env := flag.String("env", "", "The environment that is being deployed into.")

	confirm := flag.Bool("confirm", true, "Suppress confirmation prompt")

	flag.Parse()

	config := Config{
		Manifest: *manifest,
		Ami: *ami,
		Env: *env,
		Confirm: *confirm,
	}

	return config
}

func (b Builder) CheckValidation()  {
	//Check manifest file
	if len(b.Config.Manifest) == 0 || ! fileExists(b.Config.Manifest) {
		error_logging(NO_MANIFEST_EXISTS)
	}
}