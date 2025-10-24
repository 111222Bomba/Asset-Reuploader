package roblox

import (
	"bytes" // EKLENDİ
	"encoding/json" // EKLENDİ
	"errors"
	"fmt"
	"io"
	"mime/multipart" // EKLENDİ
	"net/http"
	"strings"
	"sync"
	"time"
)

const cookieWarning = "WARNING:-DO-NOT-SHARE-THIS.--Sharing-this-will-allow-someone-to-log-in-as-you-and-to-steal-your-ROBUX-and-items."
const assetUploadURL = "https://publish.roblox.com/v1/audio" // Roblox Ses Yükleme API'si

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

// ... [NewClient, SetCookie, GetToken, SetToken ve DoRequest fonksiyonlarının geri kalanı aynı kalır] ...

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

	// authenticate fonksiyonunun var olduğunu varsayıyoruz
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

func (c *Client) GetAssetLocation(id int64, assetTypeID int32) (string, error) {
    return fmt.Sprintf("https://assetdelivery.roblox.com/v1/asset/?id=%d", id), nil
}


// KRİTİK EKLENTİ: SES YÜKLEME FONKSİYONUNUN TAM UYGULAMASI
func (c *Client) ReuploadSound(body io.Reader, placeID int64) (int64, error) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 1. Ses dosyasını (audioFile) form alanına ekle
	part, err := writer.CreateFormFile("audioFile", "sound.ogg")
	if err != nil {
		return 0, fmt.Errorf("error creating form file: %w", err)
	}

	// io.Reader'ı (body) part'a kopyala
	if _, err = io.Copy(part, body); err != nil {
		return 0, fmt.Errorf("error copying audio data: %w", err)
	}

	// 2. Diğer zorunlu alanları ekle (placeId)
	if err = writer.WriteField("placeId", fmt.Sprintf("%d", placeID)); err != nil {
		return 0, fmt.Errorf("error writing placeId field: %w", err)
	}
	
	// Yükleme sırasında varsayılan bir isim kullanıyoruz
	if err = writer.WriteField("name", "Reuploaded Sound"); err != nil {
		return 0, fmt.Errorf("error writing name field: %w", err)
	}

	writer.Close() // Form data'yı kapat

	// 3. API İsteğini Yap
	req, err := http.NewRequest("POST", assetUploadURL, &requestBody)
	if err != nil {
		return 0, err
	}

	// Zorunlu Başlıklar
	req.Header.Set("Cookie", ".ROBLOSECURITY="+c.Cookie)
	req.Header.Set("X-CSRF-TOKEN", c.GetToken()) // token'ınızın güncel olduğunu varsayıyoruz
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.DoRequest(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	// 4. Yanıtı Ayrıştır (Yeni Ses ID'sini Çıkar)
	var uploadResponse struct {
		AssetId int64 `json:"assetId"`
		// Diğer alanlar olabilir, ancak sadece AssetId gerekli
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&uploadResponse); err != nil {
		return 0, fmt.Errorf("failed to decode upload response: %w", err)
	}

	if uploadResponse.AssetId == 0 {
		return 0, errors.New("roblox returned a success status but AssetId is 0. Check API response body for details.")
	}

	return uploadResponse.AssetId, nil
}
