package main

import (
	"fmt"
	"log"

	"net/url"

	"encoding/json"

	"github.com/MariaTerzieva/gotumblr"
	"github.com/PuerkitoBio/goquery"
	"github.com/garyburd/go-oauth/oauth"
	"github.com/joho/godotenv"
	"github.com/skratchdot/open-golang/open"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

func main() {
	dstDir := "imgs"
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

	dashboard := client.Dashboard(map[string]string{"limit": "10"})
	if len(dashboard.Posts) == 0 {
		return
	}

	photoUrls := getImageUrls(convertJsonToPhotoPosts(dashboard.Posts))
	downloadPhotos(photoUrls, dstDir)

	response := client.Posts("awesome_blog", "video", map[string]string{})

	videoUrls, err := getVideoUrls(convertJsonToVideoPosts(response.Posts))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(videoUrls)
}

func convertJsonToVideoPosts(jsonPosts []json.RawMessage) []gotumblr.VideoPost {
	var videoPosts []gotumblr.VideoPost
	var videoPost gotumblr.VideoPost
	for _, post := range jsonPosts {
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

func getVideoUrls(videoPosts []gotumblr.VideoPost) ([]string, error) {
	var videoUrls []string
	for _, post := range videoPosts {
		if post.PostType != "video" {
			continue
		}

		maxSizePlayer := post.Player[0]
		for _, player := range post.Player {
			if maxSizePlayer.Width < player.Width {
				maxSizePlayer = player
			}
		}

		reader := strings.NewReader(maxSizePlayer.Embed_code)
		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			return nil, err
		}

		videoDOM := doc.Find("video")
		if videoDOM != nil {
			sourceDOM := videoDOM.Find("source")
			if source, ok := sourceDOM.Attr("src"); ok {
				if videoType, ok := sourceDOM.Attr("type"); ok {
					splitedVideoType := strings.Split(videoType, "/")
					videoExt := splitedVideoType[len(splitedVideoType)-1]
					videoUrls = append(videoUrls, fmt.Sprintf("%s.%s", source, videoExt))
				}
			}
		}
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

func getImageFileName(imageUrl string) (string, error) {
	parsedImageUrl, err := url.Parse(imageUrl)
	if err != nil {
		return "", err
	}
	splitedImagePath := strings.Split(parsedImageUrl.Path, "/")
	return splitedImagePath[len(splitedImagePath)-1], nil
}

func isExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func downloadPhotos(photoUrls []string, dstDir string) error {
	for _, photoUrl := range photoUrls {
		photoFileName, err := getImageFileName(photoUrl)
		if err != nil {
			return err
		}
		ok, err := download(photoUrl, dstDir)
		if err != nil {
			return err
		}

		if ok {
			fmt.Println("image is downloaded from " + photoUrl + " to " + path.Join(dstDir, photoFileName))
		}
	}
	return nil
}

func download(imageUrl string, dstDir string) (bool, error) {
	imageFileName, err := getImageFileName(imageUrl)
	if err != nil {
		return false, err
	}

	if !isExist(dstDir) {
		if err := os.MkdirAll(dstDir, 0777); err != nil {
			return false, err
		}
	}

	if isExist(path.Join(dstDir, imageFileName)) {
		return false, nil
	}

	response, err := http.Get(imageUrl)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	file, err := os.Create(path.Join(dstDir, imageFileName))
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
