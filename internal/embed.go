package internal

import _ "embed"

// We embed two files from your project:
//   alpine_versions.json
//   alpine.apkovl.tar.gz

//go:embed data/alpine_versions.json
var VersionsData []byte

//go:embed data/alpine.apkovl.tar.gz
var ApkovlTar []byte
