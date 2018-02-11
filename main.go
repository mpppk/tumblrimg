package main

import (
	"fmt"
	"log"

	"net/url"

	"encoding/json"

	"github.com/MariaTerzieva/gotumblr"
	"github.com/garyburd/go-oauth/oauth"
	"github.com/joho/godotenv"
	"github.com/skratchdot/open-golang/open"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type VideoPost struct {
	gotumblr.BasePost
	VideoUrl string `json:"video_url"`
}

func main() {
	photoDstDir := "imgs"
	videoDstDir := "videos"
	postNumPerBlog := 500

	fetchGlobalOffset := 0
	if len(os.Args) > 1 {
		num, err := strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		fetchGlobalOffset = num
	}

	maxBlogNum := 200
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	consumerKey := os.Getenv("CONSUMER_KEY")
	consumerSecret := os.Getenv("CONSUMER_SECRET")
	oauthToken := os.Getenv("OAUTH_TOKEN")
	oauthSecret := os.Getenv("OAUTH_SECRET")

	client := gotumblr.NewTumblrRestClient(
		consumerKey,
		consumerSecret,
		oauthToken,
		oauthSecret,
		"callback_url",
		"http://api.tumblr.com",
	)

	blogOffset := 0
	var blogNames []string
	for blogOffset <= maxBlogNum {
		blogs := client.Following(map[string]string{"offset": fmt.Sprint(blogOffset)}).Blogs

		if len(blogs) == 0 {
			fmt.Println("blog num zero")
			break
		}
		for _, blog := range blogs {
			blogNames = append(blogNames, blog.Name)
		}
		blogOffset += 20
	}

	requestCount := 0
	for i, blogName := range blogNames {
		fmt.Printf("---- fetch from %s %d/%d----\n", blogName, i, len(blogNames))
		fetchNum := fetchGlobalOffset
		for fetchNum <= postNumPerBlog+fetchGlobalOffset {
			opt := map[string]string{"offset": fmt.Sprint(fetchNum)}
			photoRes := client.Posts(blogName, "photo", opt)
			requestCount++
			photoUrls := getImageUrls(convertJsonToPhotoPosts(photoRes.Posts))
			log.Printf("%d photo URLs are found on %s %d-%d / %d request: %d",
				len(photoUrls), blogName, fetchNum, fetchNum+20, postNumPerBlog+fetchGlobalOffset, requestCount)
			if len(photoUrls) == 0 {
				time.Sleep(4000 * time.Millisecond)
				break
			}
			err = downloadFiles(photoUrls, path.Join(photoDstDir, blogName))
			if err != nil {
				log.Print(err)
				break
			}
			fetchNum += 20
			time.Sleep(4000 * time.Millisecond)
		}

		fetchNum = fetchGlobalOffset
		for fetchNum <= postNumPerBlog+fetchGlobalOffset {
			opt := map[string]string{"offset": fmt.Sprint(fetchNum)}
			videoRes := client.Posts(blogName, "video", opt)
			requestCount++
			videoUrls, err := getVideoUrls(convertJsonToVideoPosts(videoRes.Posts))
			if err != nil {
				log.Print(err)
			}
			log.Printf("%d video URLs are found on %s %d-%d / %d request: %d",
				len(videoUrls), blogName, fetchNum, fetchNum+20, postNumPerBlog+fetchGlobalOffset, requestCount)
			if len(videoUrls) == 0 {
				time.Sleep(4000 * time.Millisecond)
				break
			}
			err = downloadFiles(videoUrls, path.Join(videoDstDir, blogName))
			if err != nil {
				log.Print(err)
			}
			fetchNum += 20
			time.Sleep(4000 * time.Millisecond)
		}
	}
}

func convertJsonToVideoPosts(jsonPosts []json.RawMessage) []VideoPost {
	var videoPosts []VideoPost
	//var videoPost gotumblr.VideoPost
	var videoPost VideoPost
	for _, post := range jsonPosts {
		//fmt.Println(fmt.Sprintf("%s", post))
		json.Unmarshal(post, &videoPost)
		if videoPost.PostType != "video" {
			continue
		}
		videoPosts = append(videoPosts, videoPost)
	}
	return videoPosts
}

func convertJsonToPhotoPosts(jsonPosts []json.RawMessage) []gotumblr.PhotoPost {
	var photoPosts []gotumblr.PhotoPost
	var photoPost gotumblr.PhotoPost
	for _, post := range jsonPosts {
		json.Unmarshal(post, &photoPost)
		if photoPost.PostType != "photo" {
			continue
		}
		photoPosts = append(photoPosts, photoPost)
	}
	return photoPosts
}

func getVideoUrls(videoPosts []VideoPost) ([]string, error) {
	var videoUrls []string
	for _, post := range videoPosts {
		if post.PostType != "video" {
			continue
		}
		videoUrls = append(videoUrls, post.VideoUrl)
	}
	return videoUrls, nil
}

func getImageUrls(photoPosts []gotumblr.PhotoPost) []string {
	var photoUrls []string
	for _, post := range photoPosts {
		if post.PostType != "photo" {
			continue
		}

		for _, photo := range post.Photos {
			maxSizeUrl := getMaxSizeUrl(photo)
			photoUrls = append(photoUrls, maxSizeUrl)
		}
	}
	return photoUrls
}

func getMaxSizeUrl(photo gotumblr.PhotoObject) string {
	maxSize := photo.Alt_sizes[0]
	for _, size := range photo.Alt_sizes {
		if maxSize.Height < size.Height {
			maxSize = size
		}
	}
	return maxSize.Url
}

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

func downloadFiles(fileUrls []string, dstDir string) error {
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

func authorize() {
	oauthClient := &oauth.Client{
		Credentials: oauth.Credentials{
			Token:  "xzORqsOREcMl19OIQjbgl3pBzfqlYUrqU4LzwZLkCEkqt2baSE",
			Secret: "8xOEM1eThFDtkDyluDK5wKZK9LBn3Cm8l5wzuR0dZTdXRNaFWm",
		},
		TemporaryCredentialRequestURI: "http://www.tumblr.com/oauth/request_token",
		ResourceOwnerAuthorizationURI: "http://www.tumblr.com/oauth/authorize",
		TokenRequestURI:               "http://www.tumblr.com/oauth/access_token",
	}

	scope := url.Values{"scope": {"read_public,write_public,read_private,write_private"}}

	tempCredentials, err := oauthClient.RequestTemporaryCredentials(nil, "", scope)
	if err != nil {
		log.Fatal("RequestTemporaryCredentials:", err)
	}

	u := oauthClient.AuthorizationURL(tempCredentials, nil)
	fmt.Printf("1. Go to %s\n2. Authorize the application\n3. Enter verification code:\n", u)
	open.Run(u)

	var code string
	fmt.Scanln(&code)

	fmt.Println("InputCode: ", code)

	tokenCard, _, err := oauthClient.RequestToken(nil, tempCredentials, code)
	if err != nil {
		log.Fatal("RequestToken:", err)
	}

	fmt.Println("Token: ", tokenCard.Token)
	fmt.Println("Secret: ", tokenCard.Secret)
}
