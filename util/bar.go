package util

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gonejack/uiprogress"
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
	bar *uiprogress.Bar
}

func (b *Bar) Write(p []byte) (int, error) {
	n := len(p)
	_ = b.bar.Set(b.bar.Current() + int64(n))

	if b.bar.Current() >= b.bar.Total {
		b.Stop()
	}

	return n, nil
}

func NewDownloadBar(url string, total int64) *Bar {
	pro := uiprogress.New()

	bar := pro.AddBar(total)
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return url
	}).AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%s (%s / %s)", b.CompletedPercentString(), humanize.Bytes(uint64(b.Current())), humanize.Bytes(uint64(b.Total)))
	})

	pro.Start()

	return &Bar{Progress: pro, bar: bar}
}
