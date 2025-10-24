package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kartFr/Asset-Reuploader/internal/app/assets"
	"github.com/kartFr/Asset-Reuploader/internal/app/request"
	"github.com/kartFr/Asset-Reuploader/internal/app/response"
	"github.com/kartFr/Asset-Reuploader/internal/color"
	"github.com/kartFr/Asset-Reuploader/internal/files"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var CompatiblePluginVersion = ""

func getOutputFileName(reuploadType string) string {
	t := time.Now()
	return fmt.Sprintf("Output_%s_%s.json", reuploadType, t.Format("2006-01-02_15-04-05"))
}

func serve(c *roblox.Client) error {
	var exportedJSONName string
	var exportJSON bool
	var busy bool
	finished := true

	respHistory := make([]response.ResponseItem, 0)
	resp := response.New(func(i response.ResponseItem) {
		if exportJSON {
			respHistory = append(respHistory, i)

			j, err := json.Marshal(respHistory)
			if err != nil {
				log.Fatal(err)
			}

			if err := files.Write(exportedJSONName, string(j)); err != nil {
				log.Fatal(err)
			}
		}
	})

	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if resp.Len() == 0 && !busy {
			w.Write([]byte("done"))
			return
		}

		// HATA DÜZELTME 1: Orijinal fonksiyon adı (Send) kullanıldı
		resp.Send(w) 
	})

	http.HandleFunc("POST /reupload", func(w http.ResponseWriter, r *http.Request) {
		if busy || !finished {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		var req request.RawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			color.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if CompatiblePluginVersion != "" && req.PluginVersion != CompatiblePluginVersion {
			w.WriteHeader(http.StatusConflict)
			return
		}
        
        // KRİTİK DÜZELTME: Sound kısıtlaması kaldırıldı!
		if req.AssetType == "Mesh" { 
			w.WriteHeader(http.StatusUnauthorized) 
			return
		}

		if exists := assets.DoesModuleExist(req.AssetType); !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		startReupload, err := assets.NewReuploadHandlerWithType(req.AssetType, c, &req, resp)
		if err != nil {
			color.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if exportJSON = req.ExportJSON; exportJSON {
			exportedJSONName = getOutputFileName(req.AssetType)
		}

		busy = true
		finished = false

		go func() {
			start := time.Now()
			err := startReupload()
			busy = false
			if err != nil {
				finished = true
				color.Error.Println("Failed to start reuploading: ", err)
				return
			}

			duration := time.Since(start)
			fmt.Printf("Reuploading took %d hours, %d minutes, %d seconds.\n", duration/time.Hour, (duration/time.Minute)%60, (duration/time.Second)%60)
			
			// HATA DÜZELTME 2: Orijinal fonksiyon adı (SendDone) kullanıldı
			resp.SendDone() 
			finished = true
		}()
	})

	return http.ListenAndServe(":"+port, nil)
}