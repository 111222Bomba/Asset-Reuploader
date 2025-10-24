package assets

import (
	"fmt"
	"io"
	"net/http"

	"github.com/111222Bomba/Asset-Reuploader/internal/app/request"
	"github.com/111222Bomba/Asset-Reuploader/internal/app/response"
	"github.com/111222Bomba/Asset-Reuploader/internal/roblox"
)

type soundReuploader struct {
	c      *roblox.Client
	req    *request.RawRequest
	resp   *response.Response
	assets map[int64]string
}

func NewSoundReuploader(c *roblox.Client, req *request.RawRequest, resp *response.Response) (ReuploadHandler, error) {
	assets, err := c.GetAssets(req.IDs, "Sound")
	if err != nil {
		return nil, err
	}

	return &soundReuploader{
		c:      c,
		req:    req,
		resp:   resp,
		assets: assets,
	}, nil
}

func (s *soundReuploader) Reupload() error {
	for oldID, assetURL := range s.assets {
		s.resp.AddItem(response.ResponseItem{
			OldID: oldID,
			NewID: 0,
		})

		resp, err := http.Get(assetURL)
		if err != nil {
			fmt.Println("Failed to download sound file: ", err)
			continue
		}

		newID, err := s.c.ReuploadSound(resp.Body, s.req.PlaceID)
		if err != nil {
			fmt.Println(err)
			resp.Body.Close()
			continue
		}

		s.resp.AddItem(response.ResponseItem{
			OldID: oldID,
			NewID: newID,
		})

		resp.Body.Close()
	}

	return nil
}

func init() {
	Modules["Sound"] = func(c *roblox.Client, req *request.RawRequest, resp *response.Response) (ReuploadHandler, error) {
		return NewSoundReuploader(c, req, resp)
	}
}