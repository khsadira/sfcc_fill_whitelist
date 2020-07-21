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
	//"strings"
	//"time"
)

type slotConfigurationSearchResult struct {
	Hits []struct {
		ResourceState string `json:"_resource_state"`
		SlotContent struct {
			SlotType string `json:"_type"`
			ContentType string `json:"type"`
		} `json:"slot_content"`
		ContextID string `json:"context_id"`
		Context string `json:"context"`
		SlotID string `json:"slot_id"`
	} `json:"hits"`
	Total int `json:"total"`
}

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
	clientID = "86730ca8-1b08-40ba-ace1-284d18729bdb"//os.Getenv(clientID)
	clientPW = "7Wv&75r$I8"//os.Getenv(clientPW)
	if clientID == "" || clientPW == "" {
		return "", errors.New("client_id and client_pw are empty")
	}
	token := askToken(clientID, clientPW)
	if token == "" {
		return "", errors.New("token is empty")
	}
	return token, nil
}

func reworkSiteID(realmID string) []string {
	if realmID == "EMEA" {
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
			"cn_uplaypc"}
	} else if (realmID == "NCSA") {
		return []string {
			"br_ubisoft",
			"br_uplaypc",
			"ca_south_park",
			"ca_ubisoft",
			"ca_uplaypc",
			"us_south_park",
			"us_ubisoft",
			"us_uplaypc"}
	} else if (realmID == "SANDBOX_SITE") {
		return []string {
			"fr_ubisoft",
			"fr_uplaypc",
			"us_ubisoft",
			"us_uplaypc"}
	}
	return []string{}
}

const host = "store-dev.ubi.com"

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

func createSlotConfsPatchBuffer(slotType string, contentType string, resourceStat string) []byte {
	return []byte(`
	{
		"_resource_state" : "` + resourceStat + `",
		   "slot_content": {
			  "_type": "` + slotType + `",
			  "type": "` + contentType + `"
			},
			  "c_soc_whitelist": true
	}
	`)
}

func cleanSlotsConfiguration(siteID string, systemObjectID string, token string) {
	queryUrlSlotConfSearch := "https://" + host + "/s/-/dw/data/v19_8/sites/" + siteID + "/slot_configuration_search"
	buffer := []byte(`{
		"query": {
			"text_query": {
				"fields": [
					"id"
				],
				"search_phrase": "` + systemObjectID + `"
			}
		},
		"select": "(**)",
		"start": 0,
		"count": 200
	}
	`)

	buf, err := querySfcc("POST", queryUrlSlotConfSearch, "Bearer", token, buffer)

	if err != nil {
		// log.Println("Patch CSLOT: "+ systemObjectID + ":", err)
		// log.Println(queryUrlSlotConfSearch)
		return
	}

	var slotConfSearchResult slotConfigurationSearchResult
	json.Unmarshal(buf, &slotConfSearchResult)

	for _, hits := range slotConfSearchResult.Hits {
		contentSlotID := hits.SlotID
		context := hits.Context
		contextID := hits.ContextID
		resourceStat := hits.ResourceState
		buffer := createSlotConfsPatchBuffer(hits.SlotContent.SlotType, hits.SlotContent.ContentType, resourceStat)
		queryUrlCSlotsPatch := "https://" + host + "/s/-/dw/data/v19_8/sites/" + siteID + "/slots/" + reworkID(contentSlotID) + "/slot_configurations/" + reworkID(systemObjectID) + "?context=" + reworkID(context + "=" + contextID)
		_, err := querySfcc("PATCH", queryUrlCSlotsPatch, "Bearer", token, buffer)

		if err != nil {
			// log.Println("Patch slots confs: " + siteID + ": " + contentSlotID + ": " + systemObjectID + ":", err)
			// log.Println(queryUrlCSlotsPatch)
		} else {
			println("CSLOTS WHITELISTED:\t"+ siteID + ":\t" + contentSlotID + ":" +systemObjectID)
		}
	}
}

func cleanCampaigns(siteID string, systemObjectID string, token string) {
	queryURL := "https://" + host + "/s/-/dw/data/v19_8/sites/" + siteID + "/campaigns/" + reworkID(systemObjectID)
	_, err := querySfcc("PATCH", queryURL, "Bearer", token, []byte(`{"c_soc_whitelist": true}`))

	if err != nil {
		// log.Println("Patch campaign: "+ systemObjectID + ":", err)
		// log.Println(queryURL)
	} else {
		println("CAMPAIGN WHITELISTED:\t" + siteID + ":\t"+ systemObjectID)
	}
}

func cleanCoupons(siteID string, systemObjectID string, token string) {
	queryURL := "https://" + host + "/s/-/dw/data/v19_8/sites/" + siteID + "/coupons/" + reworkID(systemObjectID)
	_, err := querySfcc("PATCH", queryURL, "Bearer", token, []byte(`{"c_soc_whitelist": true}`))

	if err != nil {
		// log.Println("Patch coupon: "+ systemObjectID + ":", err)
		// log.Println(queryURL)
	} else {
		println("COUPON WHITELISTED:\t" + siteID + ":\t"+ systemObjectID)
	}
}

func cleanPromotions(siteID string, systemObjectID string, token string) {
	queryURL := "https://" + host + "/s/-/dw/data/v19_8/sites/" + siteID + "/promotions/" + reworkID(systemObjectID)
	_, err := querySfcc("PATCH", queryURL, "Bearer", token, []byte(`{"c_soc_whitelist": true}`))

	if err != nil {
		// log.Println("Patch promotion: "+ systemObjectID + ":", err)
		// log.Println(queryURL)
	} else {
		println("PROMOTION WHITELISTED:\t" + siteID + ":\t"+ systemObjectID)
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

	siteIDs := reworkSiteID("NCSA")	
	sheetsMap := f.GetSheetMap()

	for _, sheetID := range sheetsMap {
		rows := f.GetRows(sheetID)
		if (sheetID == "PROMOTIONS") {
			for _, row := range rows {
				for _, systemObjectID := range row {
					for _, siteID := range siteIDs {
						cleanPromotions(siteID, systemObjectID, token)
					}
				}
			}
		} else if (sheetID == "CAMPAIGNS") {
			for _, row := range rows {
				for _, systemObjectID := range row {
					for _, siteID := range siteIDs {
						cleanCampaigns(siteID, systemObjectID, token)
					}
				}
			}
		} else if (sheetID == "COUPONS") {
			for _, row := range rows {
				for _, systemObjectID := range row {
					for _, siteID := range siteIDs {
						cleanCoupons(siteID, systemObjectID, token)
					}
				}
			}
		} else if (sheetID == "CSLOTS") {
			for _, row := range rows {
				for _, systemObjectID := range row {
					for _, siteID := range siteIDs {
						cleanSlotsConfiguration(siteID, systemObjectID, token)
					}
				}
			}
		}
	}
}	