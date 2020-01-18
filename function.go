package p

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	bucketName          = "images.sonurai.com"
	firestoreCollection = "BingWallpapers"
	bingURL             = "http://www.bing.com"
)

var (
	ENMarkets = []string{
		"en-ww",
		"en-gb",
		"en-us",
	}

	nonENMarkets = []string{
		"zh-cn",
		"ja-JP",
	}
)

type PubSubMessage struct {
	Data []byte `json:"data"`
}

func UpdateWallpapers(ctx context.Context, m PubSubMessage) error {
	Start(ctx)
	return nil
}

func Start(ctx context.Context) {
	saJSON, _ := base64.StdEncoding.DecodeString(os.Getenv("FIRESTORE_SA"))
	sa := option.WithCredentialsJSON(saJSON)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		panic(err)
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		panic(err)
	}
	defer firestoreClient.Close()

	storageClient, err := app.Storage(ctx)
	if err != nil {
		panic(err)
	}

	bucket, err := storageClient.Bucket(bucketName)
	if err != nil {
		panic(err)
	}

	if _, err = bucket.Attrs(ctx); err != nil {
		panic(err)
	}

	wallpapers := make(map[string]Image)

	for _, market := range ENMarkets {
		bw, err := getData(market)
		if err != nil {
			panic(err)
		}

		for _, v := range bw.Images {
			image, err := convertToImage(v, market)
			if err != nil {
				panic(err)
			}

			if _, exists := wallpapers[image.ID]; !exists {
				wallpapers[image.ID] = *image
			}
		}
	}

	for _, market := range nonENMarkets {
		bw, err := getData(market)
		if err != nil {
			panic(err)
		}

		for _, v := range bw.Images {
			image, err := convertToImage(v, market)
			if err != nil {
				panic(err)
			}

			if _, exists := wallpapers[image.ID]; !exists {
				wallpapers[image.ID] = *image
			}
		}
	}

	// for each wallpaper, check if exists in db
	fmt.Println(len(wallpapers))
	for _, v := range wallpapers {
		if !fileExists(v.URL) {
			continue
		}
		//fmt.Printf("%+v\n", v)
		dsnap, err := firestoreClient.Collection(firestoreCollection).Doc(v.ID).Get(ctx)
		if err != nil {
			//fmt.Println(err.Error())
		}

		if status.Code(err) == codes.NotFound {
			fmt.Println("not found")
		}

		if dsnap.Exists() {
			data := dsnap.Data()

			var result Image
			mapstructure.Decode(data, &result)

			if stringInSlice(result.Market, nonENMarkets) && stringInSlice(v.Market, ENMarkets) {
				v.Date = result.Date
			} else {
				continue
			}
		}

		downloadFile(ctx, bucket, v.URL, v.Filename+".jpg")
		downloadFile(ctx, bucket, v.ThumbURL, v.Filename+"_th.jpg")

		var wallpaper map[string]interface{}
		inrec, _ := json.Marshal(v)
		json.Unmarshal(inrec, &wallpaper)

		_, err = firestoreClient.Collection(firestoreCollection).Doc(v.ID).Set(ctx, wallpaper, firestore.MergeAll)
		if err != nil {
			log.Fatalf("Failed adding: %v", err)
		}
	}
}

type BingWallpapers struct {
	Images []BingImage `json:"images"`
}

type BingImage struct {
	StartDate string `json:"startdate"`
	Copyright string `json:"copyright"`
	URLBase   string `json:"urlbase"`
	URL       string `json:"url"`
}

type Image struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
	Date      int    `json:"date"`
	Filename  string `json:"filename"`
	Market    string `json:"market"`
	FullDesc  string `json:"fullDesc"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumbUrl"`
}

func getData(market string) (*BingWallpapers, error) {
	resp, _ := http.Get("https://www.bing.com/HPImageArchive.aspx?format=js&n=10&mbl=1&mkt=" + market)
	defer resp.Body.Close()

	bw := new(BingWallpapers)
	if err := json.NewDecoder(resp.Body).Decode(bw); err != nil {
		return nil, err
	}

	return bw, nil
}

func convertToImage(bw BingImage, market string) (*Image, error) {
	fullDesc := bw.Copyright
	id := strings.Replace(bw.URLBase, "/az/hprichbg/rb/", "", 1)
	filename := strings.Replace(id, "/th?id=OHR.", "", 1)
	id = strings.Split(filename, "_")[0]

	date, err := strconv.Atoi(bw.StartDate)
	if err != nil {
		return nil, err
	}

	var copyright string
	var title string
	a := strings.Split(bw.Copyright, "(")
	if len(a) != 2 {
		// failed because of chinese chars
		s := strings.Split(bw.Copyright, "（")
		title = s[0]
		copyright = s[1]
		copyright = strings.Replace(copyright, "）", "", 1)
	} else {
		title = a[0]
		copyright = a[1]
		copyright = strings.Replace(copyright, ")", "", 1)
	}
	title = strings.TrimSpace(title)
	copyright = strings.TrimSpace(copyright)

	image := &Image{
		ID:        id,
		Title:     title,
		Copyright: copyright,
		Date:      date,
		Filename:  filename,
		Market:    market,
		FullDesc:  fullDesc,
		URL:       bingURL + bw.URLBase + "_1920x1200.jpg",
		ThumbURL:  bingURL + bw.URLBase + "_1920x1080.jpg",
	}

	return image, nil
}

func fileExists(url string) bool {
	req, _ := http.NewRequest(http.MethodHead, url, nil)
	client := http.DefaultClient
	resp, _ := client.Do(req)

	return resp.StatusCode == 200
}

func downloadFile(ctx context.Context, bucket *storage.BucketHandle, url string, name string) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	client := http.DefaultClient
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	objWriter := bucket.Object(name).NewWriter(ctx)

	_, err := io.Copy(objWriter, resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	objWriter.Close()
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
