package utils

import (
	"gopkg.in/go-mixed/go-common.v1/utils/list"
	"strings"
)

var VideoExtensions = []string{
	"mp4", "m4v", "m4p",
	"mpeg", "mpg",
	"wmv",
	"rm", "rmvb",
	"avi",
	"ts", "tp",
	"webm",
	"hevc",
	"mkv",
	"flv", "f4v",
	"mov",
	"asf",
}

func IsVideo(file string) bool {
	segments := strings.Split(file, ".")
	if len(segments) <= 0 {
		return false
	}
	ext := strings.ToLower(segments[len(segments)-1])

	return list_utils.StrIndexOf(VideoExtensions, ext, false) >= 0
}

var ImageExtensions = []string{
	"jpg", "jpeg",
	"gif",
	"png",
	"bmp",
	"tif", "tiff",
	"webp",
	"svg",
	"ico", "icon",
	"avif",
}

func IsImage(file string) bool {
	segments := strings.Split(file, ".")
	if len(segments) <= 0 {
		return false
	}
	ext := strings.ToLower(segments[len(segments)-1])
	return list_utils.StrIndexOf(ImageExtensions, ext, false) >= 0
}
