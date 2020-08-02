package version

import "fmt"

var latestVersion = "LATEST_VERSION"

type Version struct {
	Version string
}

func New() Version {
	return Version{Version: latestVersion}
}

func (v Version) Get() string {
	return v.Version
}

func (v Version) Print() error {
	fmt.Println(v.Get())
	return nil
}
