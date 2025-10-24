package sound

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/111222Bomba/Asset-Reuploader/internal/app/assets/shared/assetutils"
	"github.com/111222Bomba/Asset-Reuploader/internal/app/assets/shared/clientutils"
	"github.com/111222Bomba/Asset-Reuploader/internal/app/assets/shared/uploaderror"
	"github.com/111222Bomba/Asset-Reuploader/internal/app/context"
	"github.com/111222Bomba/Asset-Reuploader/internal/app/request"
	"github.com/111222Bomba/Asset-Reuploader/internal/app/response"
	"github.com/111222Bomba/Asset-Reuploader/internal/retry"
	"github.com/111222Bomba/Asset-Reuploader/internal/roblox/develop" // KRİTİK DÜZELTME
)

const assetTypeID int32 = 3 // Sound asset tipi

func Reupload(ctx *context.Context, r *request.Request) {
	client := ctx.Client
	logger := ctx.Logger
	pauseController := ctx.PauseController
	resp := ctx.Response

	idsToUpload := len(r.IDs)
	var idsProcessed atomic.Int32

	filter := assetutils.NewFilter(ctx, r, assetTypeID)
	
	logger.Println("Reuploading sounds...")

	newBatchError := func(amt int, m string, err any) {
		end := int(idsProcessed.Add(int32(amt)))
		start := end - amt
		logger.Error(uploaderror.NewBatch(start, end, idsToUpload, m, err))
	}

	// HATA DÜZELTME: develop.AssetInfo kullanıldı
	newUploadError := func(m string, assetInfo *develop.AssetInfo, err any) {
		newValue := idsProcessed.Add(1)
		logger.Error(uploaderror.New(int(newValue), idsToUpload, m, assetInfo, err))
	}

	// HATA DÜZELTME: develop.AssetInfo kullanıldı
	uploadAsset := func(assetInfo *develop.AssetInfo) {
		oldName := assetInfo.Name

		// 1. Asset'in indirileceği URL'i bul
		location, err := client.GetAssetLocation(assetInfo.ID, assetTypeID)
		if err != nil {
			newUploadError("Failed to get asset location", assetInfo, err)
			return
		}

		// 2. Sound dosyasını indir
		assetDataResp, err := http.Get(location)
		if err != nil {
			newUploadError("Failed to download sound file", assetInfo, err)
			return
		}
		defer assetDataResp.Body.Close()

		// 3. Sound dosyasını Roblox'a yükle
		res := <-retry.DoTask( 
			retry.NewOptions(retry.Tries(3)),
			func(try int) (int64, error) {
				pauseController.WaitIfPaused()
				if try > 1 {
					// Buraya Rate Limit bekleme mekanizması eklenebilir.
				}
				
				id, err := client.ReuploadSound(assetDataResp.Body, r.PlaceID)
				if err != nil {
					if err.Error() == "cookie expired" { 
						clientutils.GetNewCookie(ctx, r, "cookie expired")
					}
					return 0, &retry.ContinueRetry{Err: err}
				}
				return id, nil
			},
		)

		if err := res.Error; err != nil {
			assetInfo.Name = oldName
			newUploadError("Failed to upload", assetInfo, err)
			return
		}

		newID := res.Result
		newValue := idsProcessed.Add(1)
		logger.Success(uploaderror.New(int(newValue), idsToUpload, "", assetInfo, newID))
		resp.AddItem(response.ResponseItem{
			OldID: assetInfo.ID,
			NewID: newID,
		})
	}

	// Asset ID'lerini çekme ve işleme kısmı (Animasyonlardaki gibi)
	var wg sync.WaitGroup
	tasks := assetutils.GetAssetsInfoInChunks(ctx, r)
	wg.Add(len(tasks))
	
	for _, task := range tasks {
		go func(task <-chan assetutils.AssetsInfoResult) {
			defer wg.Done()
			res := <-task
			
			if err := res.Error; err != nil {
				// HATA DÜZELTME: len(res.Result.Assets) kullanıldı
				newBatchError(len(res.Result.Assets), "Failed to get assets info", err)
				return
			}
			
			// HATA DÜZELTME: res.Result.Assets'teki AssetInfo slice'ı filtrele
			filteredInfo := filter(res.Result.Assets)
			
			for _, assetInfo := range filteredInfo {
				uploadAsset(assetInfo)
			}
		}(task)
	}
	
	wg.Wait()
}
