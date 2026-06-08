package screenmate

import (
	"context"
	"fmt"
	"log"
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

func (c *Client) Status(ctx context.Context) (RoomStatus, error) {
	if err := c.login(ctx); err != nil {
		return RoomStatus{}, fmt.Errorf("login: %w", err)
	}

	if err := c.selectRoom(ctx); err != nil {
		return RoomStatus{}, fmt.Errorf("select room: %w", err)
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

// internal/screenmate/client.go

func (c *Client) TogglePower(ctx context.Context, acNumber int) error {
	if err := c.login(ctx); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	if err := c.selectRoom(ctx); err != nil {
		return fmt.Errorf("select room: %w", err)
	}

	acs, err := ParseAirConditioners(c.currentPage.Doc)
	if err != nil {
		return fmt.Errorf("parse conditioners: %w", err)
	}

	var ac AirConditioner
	found := false

	for _, item := range acs {
		if item.Number == acNumber {
			ac = item
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("air conditioner %d not found", acNumber)
	}

	if ac.ToggleTarget == "" {
		return fmt.Errorf("air conditioner %d has empty toggle target", acNumber)
	}

	log.Printf(
		"toggle ac=%d currentPower=%v target=%q pageURL=%q action=%q",
		ac.Number,
		ac.Power,
		ac.ToggleTarget,
		c.currentPage.URL,
		c.currentPage.Action,
	)

	if err := c.click(ctx, ac.ToggleTarget); err != nil {
		return fmt.Errorf("click toggle: %w", err)
	}

	after, err := ParseAirConditioners(c.currentPage.Doc)
	if err != nil {
		return fmt.Errorf("parse after toggle: %w", err)
	}

	for _, item := range after {
		if item.Number == acNumber {
			log.Printf(
				"after toggle ac=%d power=%v target=%q",
				item.Number,
				item.Power,
				item.ToggleTarget,
			)
			break
		}
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

func (c *Client) click(ctx context.Context, eventTarget string) error {
	if c.currentPage == nil {
		return fmt.Errorf("current page is empty")
	}

	form := cloneValues(c.currentPage.Values)

	form.Set("__EVENTTARGET", eventTarget)
	form.Set("__EVENTARGUMENT", "")
	form.Set("__LASTFOCUS", "")

	log.Printf(
		"toggle post eventTarget=%q roomIdPresent=%v action=%q referer=%q",
		eventTarget,
		form.Get("roomId") != "",
		c.currentPage.Action,
		c.currentPage.URL,
	)

	nextPage, err := c.postPageForm(ctx, c.currentPage.Action, form, c.currentPage.URL)
	if err != nil {
		return err
	}

	c.currentPage = nextPage

	if strings.Contains(nextPage.Body, "Correct these errors") {
		return fmt.Errorf("screenmate returned validation error")
	}

	return nil
}

func isLoginPage(body string) bool {
	return strings.Contains(body, `id="LoginForm"`) ||
		strings.Contains(body, `name="userName"`) ||
		strings.Contains(body, `name="password"`)
}
