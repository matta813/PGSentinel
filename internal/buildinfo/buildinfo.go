package buildinfo

var (
	Version   = "dev"
	CommitSHA = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	CommitSHA string `json:"commit"`
	BuildTime string `json:"buildTime"`
}

func Current() Info { return Info{Version: Version, CommitSHA: CommitSHA, BuildTime: BuildTime} }
