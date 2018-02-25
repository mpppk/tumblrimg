package img

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

func getFileNameFromUrl(fileUrl string) (string, error) {
	parsedFileUrl, err := url.Parse(fileUrl)
	if err != nil {
		return "", err
	}
	splitedFilePath := strings.Split(parsedFileUrl.Path, "/")
	return splitedFilePath[len(splitedFilePath)-1], nil
}

func isExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func DownloadFiles(fileUrls []string, dstDir string) error {
	for _, fileUrl := range fileUrls {
		downloaded, err := download(fileUrl, dstDir)
		if err != nil {
			return err
		}

		if downloaded {
			time.Sleep(2000 * time.Millisecond)
		}
	}
	return nil
}

func download(fileUrl string, dstDir string) (bool, error) {
	fileName, err := getFileNameFromUrl(fileUrl)
	if err != nil {
		return false, err
	}

	if !isExist(dstDir) {
		if err := os.MkdirAll(dstDir, 0777); err != nil {
			return false, err
		}
	}

	if isExist(path.Join(dstDir, fileName)) {
		return false, nil
	}

	log.Printf("downloading from %s to %s...", fileUrl, path.Join(dstDir, fileName))
	response, err := http.Get(fileUrl)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	file, err := os.Create(path.Join(dstDir, fileName))
	if err != nil {
		return false, err
	}
	defer file.Close()
	io.Copy(file, response.Body)
	return true, nil
}
