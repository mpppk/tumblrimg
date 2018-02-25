package main

import (
	"fmt"
	"log"

	"github.com/MariaTerzieva/gotumblr"
	"github.com/joho/godotenv"
	"github.com/mpppk/tumblrimg/img"
	"github.com/mpppk/tumblrimg/tumblr"
	"os"
	"path"
	"strconv"
	"time"
)

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
			photoUrls := tumblr.GetImageUrls(tumblr.ConvertJsonToPhotoPosts(photoRes.Posts))
			log.Printf("%d photo URLs are found on %s %d-%d / %d request: %d",
				len(photoUrls), blogName, fetchNum, fetchNum+20, postNumPerBlog+fetchGlobalOffset, requestCount)
			if len(photoUrls) == 0 {
				time.Sleep(4000 * time.Millisecond)
				break
			}

			downloadNum, err := img.DownloadFiles(photoUrls, path.Join(photoDstDir, blogName), 2000)
			if err != nil {
				log.Print(err)
				break
			}

			if downloadNum == 0 {
				time.Sleep(4000 * time.Millisecond)
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

			videoUrls, err := tumblr.GetVideoUrls(tumblr.ConvertJsonToVideoPosts(videoRes.Posts))
			if err != nil {
				log.Print(err)
			}

			log.Printf("%d video URLs are found on %s %d-%d / %d request: %d",
				len(videoUrls), blogName, fetchNum, fetchNum+20, postNumPerBlog+fetchGlobalOffset, requestCount)

			if len(videoUrls) == 0 {
				time.Sleep(4000 * time.Millisecond)
				break
			}

			downloadNum, err := img.DownloadFiles(videoUrls, path.Join(videoDstDir, blogName), 2000)
			if err != nil {
				log.Print(err)
			}

			if downloadNum == 0 {
				time.Sleep(4000 * time.Millisecond)
				break
			}

			fetchNum += 20
			time.Sleep(4000 * time.Millisecond)
		}
	}
}
