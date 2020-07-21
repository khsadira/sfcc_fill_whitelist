package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func reworkID(id string) string {
	id = url.QueryEscape(id)
	return id
}

//Get Token
type Token struct {
	AccessToken string `json:"access_token"`
}

func askToken(clientID string, clientPW string) string {
	client := &http.Client{}
	key := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientPW))
	query := fmt.Sprintf("https://account.demandware.com/dwsso/oauth2/access_token?grant_type=client_credentials")
	req, err := http.NewRequest("POST", query, nil)
	req.Header.Add("Authorization", "Basic "+key)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return ""
	}

	defer resp.Body.Close()
	buf, _ := ioutil.ReadAll(resp.Body)

	var token Token
	json.Unmarshal(buf, &token)
	return token.AccessToken
}

func getToken(clientID string, clientPW string) (string, error) {
	clientID = os.Getenv(clientID)
	clientPW = os.Getenv(clientPW)
	if clientID == "" || clientPW == "" {
		return "", errors.New("client_id and client_pw are empty")
	}
	token := askToken(clientID, clientPW)
	if token == "" {
		return "", errors.New("token is empty")
	}
	return token, nil
}

func reworkSiteID(sheetID string) []string {
	if sheetID == "GLOBAL" {
		return []string{
			"ie_uplaypc",
			"jp_ubisoft",
			"eu_ubisoft",
			"de_ubisoft",
			"at_uplaypc",
			"it_ubisoft",
			"kr_ubisoft",
			"sea_ubisoft",
			"de_uplaypc",
			"ru_uplaypc",
			"uk_ubisoft",
			"ie_ubisoft",
			"es_ubisoft",
			"nl_ubisoft",
			"it_uplaypc",
			"sea_uplaypc",
			"eu_uplaypc",
			"jp_uplaypc",
			"uk_uplaypc",
			"kr_uplaypc",
			"at_ubisoft",
			"ru_ubisoft",
			"nl_uplaypc",
			"tr_ubisoft",
			"cn_ubisoft",
			"fr_uplaypc",
			"anz_uplaypc",
			"anz_ubisoft",
			"fr_ubisoft",
			"es_uplaypc",
			"tr_uplaypc",
			"cn_uplaypc"
		}
	} else if (sheetID == "NCSA") {
		return []string {
			"br_ubisoft",
			"br_uplaypc",
			"ca_south_park",
			"ca_ubisoft",
			"ca_uplaypc",
			"us_south_park",
			"us_ubisoft",
			"us_uplaypc"
		}
	}

	sheetID = strings.ToLower(sheetID)

	siteIDs := []string{sheetID + "_ubisoft", sheetID + "_uplaypc"}

	return siteIDs
}

type promotioncampaignassignmentsearchResult struct {
	V     string `json:"_v"`
	Type  string `json:"_type"`
	Count int    `json:"count"`
	Hits  []struct {
		Type          string `json:"_type"`
		ResourceState string `json:"_resource_state"`
		CampaignID    string `json:"campaign_id"`
		Enabled       bool   `json:"enabled"`
		Link          string `json:"link"`
		PromotionID   string `json:"promotion_id"`
		Schedule      struct {
			Type string `json:"_type"`
		} `json:"schedule"`
		Description string `json:"description,omitempty"`
	} `json:"hits"`
	Query struct {
		TextQuery struct {
			Type         string   `json:"_type"`
			Fields       []string `json:"fields"`
			SearchPhrase string   `json:"search_phrase"`
		} `json:"text_query"`
	} `json:"query"`
	Select string `json:"select"`
	Start  int    `json:"start"`
	Total  int    `json:"total"`
}

type campaignStruct struct {
	V             string    `json:"_v"`
	Type          string    `json:"_type"`
	ResourceState string    `json:"_resource_state"`
	CampaignID    string    `json:"campaign_id"`
	Coupons       []string  `json:"coupons"`
	CreationDate  time.Time `json:"creation_date"`
	Enabled       bool      `json:"enabled"`
	EndDate       time.Time `json:"end_date"`
	LastModified  time.Time `json:"last_modified"`
	Link          string    `json:"link"`
	StartDate     time.Time `json:"start_date"`
	CSocWhitelist bool      `json:"c_soc_whitelist"`
}

const host = "production-na01-ubisoft.demandware.net"

/*
	SFCC CONNECTION
*/

func querySfcc(method string, query string, auth string, token string, body []byte) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, query, bytes.NewBuffer(body))

	if err != nil {
		return []byte(""), err
	}

	if auth == "Bearer" {
		req.Header.Add("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Add("Content-Type", "application/json")
		}
	} else if auth == "Basic" {
		req.Header.Add("Authorization", "Basic "+token)
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return nil, errors.New(resp.Status)
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	return buf, err
}

func whitelistPromotions(queryURL string, campaignID string, token string, ch chan bool) {
	queryPromoAssignmentUrl := queryURL + "/promotion_campaign_assignment_search"

	body := fmt.Sprintf(`{"query":{"text_query":{"fields":["campaign_id"],"search_phrase":"%s"}},"select": "(**)","start": 0,"count":200}`, campaignID)
	respBody, err := querySfcc("POST", queryPromoAssignmentUrl, "Bearer", token, []byte(body))

	if err != nil {
		log.Println("getPromo:"+campaignID+":", err)
		log.Println(queryPromoAssignmentUrl)
		ch <- true
		return
	}

	var resp promotioncampaignassignmentsearchResult
	err = json.Unmarshal(respBody, &resp)

	if err != nil {
		log.Println("whitelistPromotions: unmarshal error")
		ch <- true
		return
	}

	queryPromoUrl := queryURL + "/promotions/"
	for _, hit := range resp.Hits {
		queryPromoURI := queryPromoUrl + reworkID(hit.PromotionID)
		_, err := querySfcc("PATCH", queryPromoURI, "Bearer", token, []byte(`{"c_soc_whitelist": true}`))

		if err != nil {
			log.Println("Patch Promo:"+hit.PromotionID+":", err)
			log.Println(queryPromoURI)

			ch <- true
			return
		}

		println("PROMOTION:\t" + hit.PromotionID)
	}

	ch <- true
}

func whitelistSO(siteID string, campaignID string, token string) {
	queryURL := "https://" + host + "/s/-/dw/data/v19_8/sites/" + siteID

	queryCampaignURL := queryURL + "/campaigns/" + reworkID(campaignID)

	respBody, err := querySfcc("PATCH", queryCampaignURL, "Bearer", token, []byte(`{"c_soc_whitelist": true}`))

	if err != nil {
		println("Patch Campaign:"+campaignID+":", err)
		log.Println(queryCampaignURL)

		return
	}

	var resp campaignStruct
	err = json.Unmarshal(respBody, &resp)

	if err != nil {
		log.Println("whitelistCampaign: unmarshal error")
		return
	}

	println("CAMPAIGN:\t" + resp.CampaignID)

	ch := make(chan bool, 1)
	go whitelistPromotions(queryURL, campaignID, token, ch)

	queryCouponURL := queryURL + "/coupons/"
	for _, couponID := range resp.Coupons {
		queryCouponURI := queryCouponURL + reworkID(couponID)
		_, err = querySfcc("PATCH", queryCouponURI, "Bearer", token, []byte(`{"c_soc_whitelist": true}`))

		if err != nil {
			log.Println("Patch Coupon:"+couponID+":", err)
			log.Println(queryCouponURI)

			return
		}

		println("COUPON:\t\t" + couponID)
	}
	for i := 0; i < 1; i++ {
		<-ch
	}
}

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		println("script usage: [script_name] file.xlsx")
		return
	}
	f, err := excelize.OpenFile(args[0])

	if err != nil {
		fmt.Println(err)
		return
	}

	token, err := getToken("SFCC_CLIENT_ID", "SFCC_CLIENT_PW")

	if err != nil {
		fmt.Println(err)
		return
	}

	sheetsMap := f.GetSheetMap()
	for _, sheetID := range sheetsMap {
		siteIDs := reworkSiteID(sheetID)
		rows := f.GetRows(sheetID)
		for _, row := range rows {
			for _, campaignID := range row {
				for _, siteID := range siteIDs {
					println("____SITE:" + siteID + "____")
					whitelistSO(siteID, campaignID, token)
					println()
				}
			}
			println()
		}
	}
}
