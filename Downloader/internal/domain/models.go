package domain

type File struct{
	ID int
	URL string
	Content []byte
	ErrorCode string
}

type DownloadTask struct{
	ID int
	Status string
	Files []File
}

type Repository interface{
	GetFile(int, int)([]byte, error)
	TaskCreation(DownloadTask) (int, error)
	RecieveDownload(int) (DownloadTask, error)
	UpdateStatus(int, string) error
	SaveFile(int, int, string, []byte) error
}

