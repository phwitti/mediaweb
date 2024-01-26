package main

import "embed"

//go:embed templates/icon_image.png
var embedImageIconBytes []byte

//go:embed templates/icon_video.png
var embedVideoIconBytes []byte

//go:embed templates/icon_folder.png
var embedFolderIconBytes []byte

//go:embed templates/logo.ico templates/index.html
var embedStaticContent embed.FS
