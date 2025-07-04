package service

type FileService interface {
	Upload(fileName string, fileContent []byte)
	Download(fileName string)
}
