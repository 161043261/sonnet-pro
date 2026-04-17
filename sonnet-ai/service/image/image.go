package image

import (
	"errors"
	"mime/multipart"
)

func RecognizeImage(file *multipart.FileHeader) (string, error) {
	return "", errors.New("Image recognition feature has been removed")
}
