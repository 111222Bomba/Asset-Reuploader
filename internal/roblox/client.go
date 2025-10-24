package roblox

import (
	"errors"
	"fmt" // GetAssetLocation metodunda kullanıldı
	"io" // <-- KRİTİK DÜZELTME: Sound modülünün ihtiyacı olan import
	"net/http"
	"strings"
	"sync"
	"time"
)

const cookieWarning = "WARNING:-DO-NOT-SHARE-THIS.--Sharing-this-will-allow-someone-to-log-in-as-you-and-to-steal-your-ROBUX-and-items."

var (
	ErrNoWarning = errors.New("include the .ROBLOSECURITY warning")
)

type Client struct {
	Cookie   string
	UserInfo UserInfo

	httpClient *http.Client

	token      string
	tokenMutex sync.RWMutex
}

func NewClient(cookie string) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if err := c.SetCookie(cookie); err != nil {
		return c, err
	}

	return c, nil
}

func (c *Client) SetCookie(cookie string) error {
	c.Cookie = strings.TrimSpace(cookie)

	if !strings.Contains(cookie, cookieWarning) {
		return ErrNoWarning
	}

	userInfo, err := authenticate(c, cookie)
	if err != nil {
		return err
	}

	c.UserInfo = userInfo
	c.Cookie = cookie
	return nil
}

func (c *Client) GetToken() string {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.token
}

func (c *Client) SetToken(s string) {
	c.tokenMutex.Lock()
	c.token = s
	c.tokenMutex.Unlock()
}

func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// KRİTİK DÜZELTME: Sound modülü tarafından çağrılır.
func (c *Client) GetAssetLocation(id int64, assetTypeID int32) (string, error) {
    // Bu, Roblox'ta varlık indirme URL'sinin yaygın formatıdır.
    return fmt.Sprintf("https://assetdelivery.roblox.com/v1/asset/?id=%d", id), nil
}

// KRİTİK DÜZELTME: Sound dosyasını yüklemek için gereklidir.
func (c *Client) ReuploadSound(body io.Reader, placeID int64) (int64, error) {
    // BURAYA ORİJİNAL KODUNUZUN ReuploadSound İÇERİĞİNİ KOYUN
    return 0, nil
}
