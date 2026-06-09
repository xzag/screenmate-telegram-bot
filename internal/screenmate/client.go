package screenmate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type Client struct {
	baseURL  string
	username string
	password string
	roomID   string

	http *http.Client

	currentPage *PageForm
}

func NewClient(baseURL, username, password, roomID string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		roomID:   roomID,
		http: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
	}, nil
}

func (c *Client) Reset() {
	jar, _ := cookiejar.New(nil)

	c.http.Jar = jar
	c.currentPage = nil
}

func (c *Client) ensureRoomPage(ctx context.Context) error {
	if c.currentPage != nil && isRoomControlPage(c.currentPage) {
		return nil
	}

	if err := c.login(ctx); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	if err := c.selectRoom(ctx); err != nil {
		return fmt.Errorf("select room: %w", err)
	}

	if !isRoomControlPage(c.currentPage) {
		return fmt.Errorf("room control page not reached")
	}

	return nil
}

func (c *Client) Status(ctx context.Context) (RoomStatus, error) {
	if err := c.ensureRoomPage(ctx); err != nil {
		return RoomStatus{}, err
	}

	if err := c.refreshRoomPage(ctx); err != nil {
		c.Reset()

		if err := c.ensureRoomPage(ctx); err != nil {
			return RoomStatus{}, fmt.Errorf("relogin after refresh failed: %w", err)
		}
	}

	acs, err := ParseAirConditioners(c.currentPage.Doc)
	if err != nil {
		return RoomStatus{}, fmt.Errorf("parse air conditioners: %w", err)
	}

	return RoomStatus{
		RoomID:          c.roomID,
		AirConditioners: acs,
		UpdatedAt:       time.Now(),
	}, nil
}

func (c *Client) TogglePower(ctx context.Context, acNumber int) error {
	if err := c.ensureRoomPage(ctx); err != nil {
		return err
	}

	acs, err := ParseAirConditioners(c.currentPage.Doc)
	if err != nil {
		return fmt.Errorf("parse conditioners: %w", err)
	}

	var target string

	for _, ac := range acs {
		if ac.Number == acNumber {
			target = ac.ToggleTarget
			break
		}
	}

	if target == "" {
		return fmt.Errorf("air conditioner %d not found", acNumber)
	}

	if err := c.postBack(ctx, target); err != nil {
		c.Reset()

		if err := c.ensureRoomPage(ctx); err != nil {
			return fmt.Errorf("relogin after toggle failed: %w", err)
		}

		return fmt.Errorf("toggle failed, session was reset: %w", err)
	}

	return nil
}

func (c *Client) login(ctx context.Context) error {
	loginURL := c.baseURL + "/LoginPage.aspx"

	page, err := c.getPageForm(ctx, loginURL)
	if err != nil {
		return err
	}

	form := cloneValues(page.Values)

	form.Set("__EVENTTARGET", "")
	form.Set("__EVENTARGUMENT", "")
	form.Set("__LASTFOCUS", "")
	form.Set("userName", c.username)
	form.Set("password", c.password)
	form.Set("userType", "VISTA_USER")
	form.Set("loginButton", "Login")

	nextPage, err := c.postPageForm(ctx, page.Action, form, page.URL)
	if err != nil {
		return err
	}

	if isLoginPage(nextPage.Body) {
		return fmt.Errorf("login failed: still on login page, url=%s", nextPage.URL)
	}

	c.currentPage = nextPage
	return nil
}

func (c *Client) selectRoom(ctx context.Context) error {
	if c.currentPage == nil {
		return fmt.Errorf("current page is empty: login first")
	}

	page := c.currentPage
	form := cloneValues(page.Values)

	form.Set("__EVENTTARGET", "lookUpRoomId")
	form.Set("__EVENTARGUMENT", "")
	form.Set("__LASTFOCUS", "")
	form.Set("roomId", c.roomID)

	// ASP.NET ImageButton coordinates.
	form.Set("lookUpRoomId.x", "11")
	form.Set("lookUpRoomId.y", "9")

	nextPage, err := c.postPageForm(ctx, page.Action, form, page.URL)
	if err != nil {
		return err
	}

	if strings.Contains(nextPage.Body, "You must specify a room ID") ||
		strings.Contains(nextPage.Body, "The specified room did not exist") ||
		strings.Contains(nextPage.Body, "Correct these errors") {
		return fmt.Errorf("room selection failed")
	}

	c.currentPage = nextPage
	return nil
}

func (c *Client) postBack(ctx context.Context, eventTarget string) error {
	if c.currentPage == nil {
		return fmt.Errorf("current page is empty")
	}

	form := cloneValues(c.currentPage.Values)

	form.Set("__EVENTTARGET", eventTarget)
	form.Set("__EVENTARGUMENT", "")
	form.Set("__LASTFOCUS", "")

	nextPage, err := c.postPageForm(ctx, c.currentPage.Action, form, c.currentPage.URL)
	if err != nil {
		return err
	}

	c.currentPage = nextPage

	if isLoginPageBody(nextPage.Body) || isRoomLookupPageBody(nextPage.Body) {
		return fmt.Errorf("session expired")
	}

	if !isRoomControlPage(nextPage) {
		return fmt.Errorf("unexpected page after postback")
	}

	return nil
}

func (c *Client) refreshRoomPage(ctx context.Context) error {
	if c.currentPage == nil {
		return fmt.Errorf("current page is empty")
	}

	form := cloneValues(c.currentPage.Values)

	form.Set("__EVENTTARGET", "Refresh:Refresh")
	form.Set("__EVENTARGUMENT", "")
	form.Set("__LASTFOCUS", "")

	nextPage, err := c.postPageForm(ctx, c.currentPage.Action, form, c.currentPage.URL)
	if err != nil {
		return err
	}

	c.currentPage = nextPage

	if isLoginPageBody(nextPage.Body) || isRoomLookupPageBody(nextPage.Body) {
		return fmt.Errorf("session expired")
	}

	if !isRoomControlPage(nextPage) {
		return fmt.Errorf("unexpected page after refresh")
	}

	return nil
}

func isLoginPage(body string) bool {
	return strings.Contains(body, `id="LoginForm"`) ||
		strings.Contains(body, `name="userName"`) ||
		strings.Contains(body, `name="password"`)
}

func isRoomControlPage(page *PageForm) bool {
	if page == nil {
		return false
	}

	return strings.Contains(page.Body, `id="dataList"`) &&
		strings.Contains(page.Body, `dataList_toggle_`)
}

func isLoginPageBody(body string) bool {
	return strings.Contains(body, `id="LoginForm"`) ||
		strings.Contains(body, `name="userName"`) ||
		strings.Contains(body, `name="password"`)
}

func isRoomLookupPageBody(body string) bool {
	return strings.Contains(body, `name="roomId"`) &&
		strings.Contains(body, `lookUpRoomId`)
}
