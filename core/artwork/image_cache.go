package artwork

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/singleton"
)

type cacheKey struct {
	artID      model.ArtworkID
	size       int
	lastUpdate time.Time
}

func (k *cacheKey) Key() string {
	return fmt.Sprintf(
		"%s.%d.%d.%d.%t",
		k.artID.ID,
		k.lastUpdate.UnixMilli(),
		k.size,
		conf.Server.CoverJpegQuality,
		conf.Server.EnableMediaFileCoverArt,
	)
}

type imageCache struct {
	cache.FileCache
}

func GetImageCache() cache.FileCache {
	return singleton.GetInstance(func() *imageCache {
		return &imageCache{
			FileCache: cache.NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
				func(ctx context.Context, arg cache.Item) (io.Reader, error) {
					r, _, err := arg.(artworkReader).Reader(ctx)
					return r, err
				}),
		}
	})
}
