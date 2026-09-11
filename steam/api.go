package steam

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.steampowered.com"

// Client инкапсулирует Steam Web API.
type Client struct {
	APIKey string
	HTTP   *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTP:   &http.Client{Timeout: 20 * time.Second},
	}
}

// get выполняет GET с повторами при временных ошибках сети.
func (c *Client) get(target string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.HTTP.Get(target)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
		}
	}
	return nil, lastErr
}

// GameServer из ответа GetServerList.
type GameServer struct {
	Addr       string `json:"addr"`
	GamePort   int    `json:"gameport"`
	QueryPort  int    `json:"queryport"`
	SteamID    string `json:"steamid"`
	Name       string `json:"name"`
	AppID      int    `json:"appid"`
	Gamedir    string `json:"gamedir"`
	Version    string `json:"version"`
	Map        string `json:"map"`
	Players    int    `json:"players"`
	MaxPlayers int    `json:"max_players"`
	Bots       int    `json:"bots"`
	Region     int    `json:"region"`
	Secure     bool   `json:"secure"`
	Dedicated  bool   `json:"dedicated"`
	OS         string `json:"os"`
	Type       string `json:"type"`
	Proxy      bool   `json:"proxy"`
}

// ServerList возвращает список серверов для appID.
func (c *Client) ServerList(appID int, limit int) ([]GameServer, error) {
	if limit <= 0 {
		limit = 100
	}
	u, _ := url.Parse(baseURL + "/IGameServersService/GetServerList/v1/")
	q := u.Query()
	q.Set("key", c.APIKey)
	q.Set("filter", fmt.Sprintf(`\appid\%d`, appID))
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	resp, err := c.get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API status %d", resp.StatusCode)
	}
	var out struct {
		Response struct {
			Servers []GameServer `json:"servers"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Response.Servers, nil
}

// WorkshopItem из поиска в мастерской.
type WorkshopItem struct {
	ID            string `json:"publishedfileid"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	CreatorName   string `json:"creator"`
	ShortDesc     string `json:"short_description"`
	TimeUpdated   int64  `json:"time_updated"`
	Subscriptions int    `json:"subscriptions"`
	Favorites     int    `json:"favorited"`
	FileSize      int64  `json:"file_size"`
}

// SearchWorkshop ищет предметы мастерской для appID по searchText.
func (c *Client) SearchWorkshop(appID int, searchText string, limit int) ([]WorkshopItem, error) {
	if limit <= 0 {
		limit = 10
	}
	u, _ := url.Parse(baseURL + "/IPublishedFileService/QueryFiles/v1/")
	q := u.Query()
	q.Set("key", c.APIKey)
	q.Set("appid", fmt.Sprintf("%d", appID))
	q.Set("query_type", "0")
	q.Set("search_text", searchText)
	q.Set("numperpage", fmt.Sprintf("%d", limit))
	q.Set("return_tags", "false")
	q.Set("return_short_description", "true")
	q.Set("return_for_sale_data", "false")
	q.Set("totalonly", "false")
	u.RawQuery = q.Encode()

	resp, err := c.get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API status %d", resp.StatusCode)
	}
	var out struct {
		Response struct {
			Total  int            `json:"total"`
			Results []WorkshopItem `json:"publishedfiledetails"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Response.Results, nil
}

// PublishedFileDetails возвращает детали предметов мастерской по их ID
// (ISteamRemoteStorage/GetPublishedFileDetails). Порциями по 100.
func (c *Client) PublishedFileDetails(ids []string) ([]WorkshopItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const chunk = 100
	var all []WorkshopItem
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		items, err := c.publishedFileDetails(ids[i:end])
		if err != nil {
			return all, err
		}
		all = append(all, items...)
	}
	return all, nil
}

func (c *Client) publishedFileDetails(ids []string) ([]WorkshopItem, error) {
	u, _ := url.Parse(baseURL + "/ISteamRemoteStorage/GetPublishedFileDetails/v1/")
	form := url.Values{}
	form.Set("itemcount", fmt.Sprintf("%d", len(ids)))
	for i, id := range ids {
		form.Set(fmt.Sprintf("publishedfileids[%d]", i), id)
	}
	resp, err := c.postForm(u.String(), form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API status %d", resp.StatusCode)
	}
	var out struct {
		Response struct {
			Results []WorkshopItem `json:"publishedfiledetails"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Response.Results, nil
}

// postForm выполняет POST с повторами при временных ошибках сети.
func (c *Client) postForm(target string, form url.Values) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.HTTP.PostForm(target, form)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
		}
	}
	return nil, lastErr
}