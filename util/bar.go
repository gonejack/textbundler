package util

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gonejack/uiprogress"
	"io"
	"math"
	"sync/atomic"
	"time"
)

func init() {
	uiprogress.Width = 40
	uiprogress.LeftEnd = '|'
	uiprogress.RightEnd = '|'
	uiprogress.Empty = '.'
	uiprogress.Fill = '='
	uiprogress.RefreshInterval = time.Second / 5
}

type Bar struct {
	*uiprogress.Progress
	*uiprogress.Bar

	reader io.Reader
	read   uint64
}

func (b *Bar) Read(p []byte) (n int, err error) {
	n, err = b.reader.Read(p)

	if n > 0 {
		next := b.Current() + int64(n)
		err := b.Set(next)
		if err == uiprogress.ErrMaxCurrentReached {
			_ = b.Set(b.Total)
		}
		atomic.AddUint64(&b.read, uint64(n))
	}

	if err != nil {
		b.Stop()
	}

	return
}

func NewDownloadBar(url string, total int64, reader io.Reader) (bar *Bar) {
	if total <= 0 {
		total = math.MaxInt64
	}

	bar = new(Bar)
	bar.reader = reader
	bar.Progress = uiprogress.New()

	bar.Bar = bar.Progress.AddBar(total)
	bar.Bar.PrependFunc(func(b *uiprogress.Bar) string {
		return url
	}).AppendFunc(func(b *uiprogress.Bar) string {
		if total == math.MaxInt64 {
			return fmt.Sprintf("(%s / Unknown)", humanize.Bytes(atomic.LoadUint64(&bar.read)))
		} else {
			return fmt.Sprintf("%s (%s / %s)", b.CompletedPercentString(), humanize.Bytes(uint64(b.Current())), humanize.Bytes(uint64(b.Total)))
		}
	})

	bar.Progress.Start()

	return
}
