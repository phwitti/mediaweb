package main

import _ "embed"

//go:embed "templates/index.html"
var embedIndexBytes []byte

//go:embed templates/icon_image.png
var embedImageIconBytes []byte

//go:embed templates/icon_video.png
var embedVideoIconBytes []byte

//go:embed templates/icon_folder.png
var embedFolderIconBytes []byte
