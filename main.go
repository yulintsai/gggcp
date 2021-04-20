package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/urfave/cli"
)

var (
	lastEventBegin = ""
	StartMsg       = "Dear，Remind you that the little bell is activated when GCP GG😉"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		{
			Name:    "watch",
			Aliases: []string{"w"},
			Usage:   "gcp",
			Action: func(c *cli.Context) error {
				fmt.Println("GGGCP is Starting ...")

				TelegramBotKey := c.Args().First()
				if TelegramBotKey == "" {
					fmt.Println("[warn] TelegramBotKey is empty")
				}

				ChatID := c.Args().Get(1)
				if ChatID == "" {
					fmt.Println("[warn] ChatID is empty")
				}

				TelUrl := "https://api.telegram.org/bot" + TelegramBotKey + "/sendMessage?chat_id=" + ChatID + "&text="
				start := &url.URL{Path: StartMsg}
				resp, _ := http.Get(TelUrl + start.String())
				resp.Body.Close()

				SlackWebHook := c.Args().Get(2)
				SlackChannel := c.Args().Get(3)
				if SlackWebHook != "" && SlackChannel != "" {
					SendToSlack(SlackWebHook, SlackChannel, "GGGCP", StartMsg)
				}

				for {
					gcpStatus, err := checkGCPStatus()
					if gcpStatus != nil {
						gt := "[warn] GCP detected new status！\n" + gcpStatus.Error() + "\nFor details, please click on the URL to view\nhttps://status.cloud.google.com/"
						fmt.Println("Full Url -> ", TelUrl+url.QueryEscape(gt))
						resp, _ := http.Get(TelUrl + url.QueryEscape(gt))

						if SlackWebHook != "" && SlackChannel != "" {
							SendToSlack(SlackWebHook, SlackChannel, "GGGCP", gt)
						}

						resp.Body.Close()

						time.Sleep(time.Duration(60) * time.Second)
					}

					if err != nil {
						et := "[warn] GGGCP has Error！\n" + err.Error()
						fmt.Println(et)
						resp, _ := http.Get(TelUrl + url.QueryEscape(et))

						if SlackWebHook != "" && SlackChannel != "" {
							SendToSlack(SlackWebHook, SlackChannel, "GGGCP", et)
						}

						resp.Body.Close()
						time.Sleep(time.Duration(60) * time.Second)
					}

					time.Sleep(time.Duration(60) * time.Second)
				}
			},
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}

func checkGCPStatus() (de error, gcp error) {
	resp, err := http.Get("https://status.cloud.google.com/incidents.json")
	if err != nil {
		fmt.Println("[warn] httpGet:", err)
		return nil, err
	}

	defer resp.Body.Close()

	type AutoGenerated struct {
		ID           string    `json:"id"`
		Number       string    `json:"number"`
		Begin        time.Time `json:"begin"`
		End          time.Time `json:"end"`
		ExternalDesc string    `json:"external_desc"`
		Updates      []struct {
			Created time.Time `json:"created"`
			When    time.Time `json:"when"`
			Text    string    `json:"text"`
			Status  string    `json:"status"`
		} `json:"updates"`
		MostRecentUpdate struct {
			Created time.Time `json:"created"`
			When    time.Time `json:"when"`
			Text    string    `json:"text"`
			Status  string    `json:"status"`
		} `json:"most_recent_update"`
		StatusImpact     string `json:"status_impact"`
		Severity         string `json:"severity"`
		ServiceName      string `json:"service_name"`
		AffectedProducts []struct {
			Title string `json:"title"`
		} `json:"affected_products"`
		URI string `json:"uri"`
	}

	body, ioReadAllErr := ioutil.ReadAll(resp.Body)
	if ioReadAllErr != nil {
		fmt.Println("[warn] ioReadAllErr:", ioReadAllErr)
		return ioReadAllErr, nil
	}

	ret := []*AutoGenerated{}

	jerr := json.Unmarshal(body, &ret)
	if jerr != nil {
		return jerr, nil
	}

	loc, loadLocationErr := time.LoadLocation("Asia/Taipei")
	if loadLocationErr != nil {
		fmt.Println("[warn] loadLocationErr:", loadLocationErr)
		return loadLocationErr, nil
	}

	// 依照時區校正
	recentTime := time.Unix(ret[0].Begin.Unix(), 0).In(loc).Format("2006-01-02 15:04:05") + " (" + loc.String() + ")"
	now := time.Now().In(loc).Format("2006-01-02 15:04:05") + " (" + loc.String() + ")"

	if lastEventBegin == "" {
		lastEventBegin = recentTime
		fmt.Println("initialize GCP recent events", lastEventBegin)

		return nil, nil
	}

	yesterday, parseDurationErr := time.ParseDuration("-24h")
	if parseDurationErr != nil {
		fmt.Println("[warn] parseDurationErr:", parseDurationErr)
		return parseDurationErr, nil
	}

	isIn24Hours := ret[0].Begin.After(time.Now().Add(yesterday))

	if lastEventBegin != recentTime {
		fmt.Println(now, "[warn] gcp has changed lastEventBegin:", lastEventBegin, " now:", recentTime)
		lastEventBegin = recentTime

		if isIn24Hours {
			fullMsg := "Started From: " + recentTime + "\nRelated Services: " + ret[0].ServiceName + "\nSeverity Level: " + ret[0].Severity

			return nil, errors.New(fullMsg)
		} else {
			fmt.Println(now, "[warn] gcp now:", recentTime, "within 24 hours, no Telegram alert will be sent")
		}
	} else {
		fmt.Println(now, "[info] gcp now status is good,", "lastEventBegin:", lastEventBegin, "isIn24Hours:", strconv.FormatBool(isIn24Hours))
	}

	return nil, nil
}

// SendToSlack
func SendToSlack(hook string, channel string, username string, text string) {
	type Payload struct {
		Text     string `json:"text"`
		Channel  string `json:"channel"`
		Username string `json:"username"`
	}

	data := Payload{
		Text:     text,
		Channel:  channel,
		Username: username,
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println("[warn] SendToSlack:", err)
	}

	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", hook, body)
	if err != nil {
		fmt.Println("[warn] SendToSlack:", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("[warn] SendToSlack:", err)
	}
	defer resp.Body.Close()
}
