package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	_ "github.com/joho/godotenv/autoload"
	"github.com/programzheng/guard/cache"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const BOOKING_SEARCH_RESULT_KEY = "booking_search_result"

var ctx = context.Background()
var rdb = cache.GetRedisClient()

func randomString() string {
	b := make([]byte, rand.Intn(10)+10)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func createMD5(secret string) string {
	// 產生模式
	hash := md5.New()

	// 轉換字串
	hash.Write([]byte(secret))

	// 最終hash結果
	bs := hash.Sum(nil)

	//將byte轉為16進制
	result := fmt.Sprintf("%x", bs)
	return result
}

func pushText(text string) {
	data := map[string]string{
		"pushId": os.Getenv("BLACK_KEY_PUSH_ID"),
		"token":  createMD5(time.Now().Format("2006-01-02")),
		"text":   text,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.Post(os.Getenv("BLACK_KEY_URI")+"/bot/line/push", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}
	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)
}

func setRedisString(text string, key string, expire string) {
	t, err := time.ParseDuration(expire)
	if err != nil {
		log.Fatal(err)
	}
	err = rdb.Set(ctx, key, text, t).Err()
	if err != nil {
		log.Fatal(err)
	}
}

func getCollyCollector(options ...func(*colly.Collector)) *colly.Collector {

	c := colly.NewCollector(options...)

	return c
}

func checkBookingStay(uri string) {
	writer, err := os.OpenFile("collector.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}

	c := getCollyCollector(
		colly.AllowedDomains("www.booking.com"),
		colly.Debugger(&debug.LogDebugger{Output: writer}),
	)

	focusUri := uri

	searchText, _ := rdb.Get(ctx, BOOKING_SEARCH_RESULT_KEY).Result()
	resultCode := ""

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", randomString())
	})

	c.OnHTML("div", func(e *colly.HTMLElement) {
		text := e.Text
		if strings.Contains(text, "住宿在您所選日期已無法預訂") {
			var re = regexp.MustCompile(`(?m)\w+%`)
			res := re.FindAllStringSubmatch(text, 1)
			resultText := ""
			for i := range res {
				resultText = res[i][0]
			}
			if searchText != resultText {
				resultCode = "update_booking_stay_percent"

				setRedisString(resultText, BOOKING_SEARCH_RESULT_KEY, "10m")
			}
		}
	})

	c.Visit(focusUri)

	switch resultCode {
	case "update_booking_stay_percent":
		pushText("可住宿的總%數已更新:\n" + os.Getenv("BOOKING_FOCUS_SHORT_URL"))
	}
}

func main() {
	focusUri := os.Getenv("BOOKING_FOCUS_URI")

	checkBookingStay(focusUri)
}
