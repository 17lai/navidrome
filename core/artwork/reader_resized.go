package artwork

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/disintegration/imaging"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/number"
)

type resizedArtworkReader struct {
	cacheKey
	a *artwork
}

func resizedFromOriginal(ctx context.Context, a *artwork, artID model.ArtworkID, size int) (*resizedArtworkReader, error) {
	r := &resizedArtworkReader{a: a}
	r.cacheKey.artID = artID
	r.cacheKey.size = size

	// Get lastUpdated from original artwork
	original, err := a.getArtworkReader(ctx, artID, 0)
	if err != nil {
		return nil, err
	}
	r.cacheKey.lastUpdate = original.LastUpdated()
	return r, nil
}

func (a *resizedArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *resizedArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	// Get artwork in original size, possibly from cache
	orig, _, err := a.a.Get(ctx, a.artID.String(), 0)
	if err != nil {
		return nil, "", err
	}

	// Keep a copy of the original data. In case we can't resize it, send it as is
	buf := new(bytes.Buffer)
	r := io.TeeReader(orig, buf)
	defer orig.Close()

	resized, origSize, err := resizeImage(r, a.size)
	log.Trace(ctx, "Resizing artwork", "artID", a.artID, "original", origSize, "resized", a.size)
	if err != nil {
		log.Warn(ctx, "Could not resize image. Will return image as is", "artID", a.artID, "size", a.size, err)
		// Force finish reading any remaining data
		_, _ = io.Copy(io.Discard, r)
		return io.NopCloser(buf), "", nil
	}
	return io.NopCloser(resized), fmt.Sprintf("%s@%d", a.artID, a.size), nil
}

func asImageReader(r io.Reader) (io.Reader, string, error) {
	br := bufio.NewReader(r)
	buf, err := br.Peek(512)
	if err != nil {
		return nil, "", err
	}
	return br, http.DetectContentType(buf), nil
}

func resizeImage(reader io.Reader, size int) (io.Reader, int, error) {
	r, format, err := asImageReader(reader)
	if err != nil {
		return nil, 0, err
	}

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, 0, err
	}

	// Preserve the aspect ratio of the image.
	var m *image.NRGBA
	bounds := img.Bounds()
	if bounds.Max.X > bounds.Max.Y {
		m = imaging.Resize(img, size, 0, imaging.Lanczos)
	} else {
		m = imaging.Resize(img, 0, size, imaging.Lanczos)
	}

	buf := new(bytes.Buffer)
	buf.Reset()
	if format == "image/png" {
		err = png.Encode(buf, m)
	} else {
		err = jpeg.Encode(buf, m, &jpeg.Options{Quality: conf.Server.CoverJpegQuality})
	}
	return buf, number.Max(bounds.Max.X, bounds.Max.Y), err
}
