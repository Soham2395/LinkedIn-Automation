package browser

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type Browser struct {
	browser *rod.Browser
	page    *rod.Page
}

func New(headless bool) (*Browser, error) {
	l := launcher.New().Headless(headless)

	if path, exists := launcher.LookPath(); exists {
		l = l.Bin(path)
	}

	url := l.MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect()

	return &Browser{
		browser: browser,
	}, nil
}

func (b *Browser) Close() {
	if b.browser != nil {
		b.browser.MustClose()
	}
}

func (b *Browser) Navigate(url string) error {
	if b.page == nil {
		b.page = b.browser.MustPage("")
	}
	return b.page.Navigate(url)
}

func (b *Browser) WaitLoad() error {
	if b.page == nil {
		return nil
	}
	return b.page.WaitLoad()
}

func (b *Browser) URL() string {
	if b.page == nil {
		return ""
	}
	return b.page.MustInfo().URL
}

func (b *Browser) TypeInto(selector, text string) error {
	el, err := b.page.Timeout(10 * time.Second).Element(selector)
	if err != nil {
		return err
	}
	return el.Input(text)
}

func (b *Browser) Click(selector string) error {
	el, err := b.page.Timeout(10 * time.Second).Element(selector)
	if err != nil {
		return err
	}
	return el.Click(proto.InputMouseButtonLeft, 1)
}

func (b *Browser) HasElement(selector string) bool {
	_, err := b.page.Timeout(3 * time.Second).Element(selector)
	return err == nil
}

func (b *Browser) TextContent(selector string) (string, error) {
	el, err := b.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return "", err
	}
	return el.Text()
}

func (b *Browser) GetCookies() ([]*proto.NetworkCookie, error) {
	return b.page.Cookies(nil)
}

func (b *Browser) SetCookies(cookies []*proto.NetworkCookieParam) error {
	return b.page.SetCookies(cookies)
}

func (b *Browser) Page() *rod.Page {
	return b.page
}

func (b *Browser) Context() context.Context {
	return b.browser.GetContext()
}
