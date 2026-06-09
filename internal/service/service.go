package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"screenmate-bot/internal/config"
	"screenmate-bot/internal/screenmate"
)

type Service struct {
	baseURL     string
	rooms       []config.RoomConfig
	maxParallel int

	sessions map[string]*roomSession
}

type roomSession struct {
	room    config.RoomConfig
	baseURL string

	mu     sync.Mutex
	client *screenmate.Client

	lastUsed time.Time
}

func New(cfg config.Config) *Service {
	maxParallel := cfg.ScreenMate.MaxParallelRooms
	if maxParallel <= 0 {
		maxParallel = 3
	}

	s := &Service{
		baseURL:     cfg.ScreenMate.BaseURL,
		rooms:       cfg.Rooms,
		maxParallel: maxParallel,
		sessions:    make(map[string]*roomSession, len(cfg.Rooms)),
	}

	for _, room := range cfg.Rooms {
		s.sessions[room.Key] = &roomSession{
			room:    room,
			baseURL: cfg.ScreenMate.BaseURL,
		}
	}

	return s
}

func (s *Service) AllRoomsStatus(ctx context.Context) []RoomView {
	views := make([]RoomView, len(s.rooms))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.maxParallel)

	for i, room := range s.rooms {
		i, room := i, room

		g.Go(func() error {
			views[i] = s.roomStatus(ctx, room)
			return nil
		})
	}

	_ = g.Wait()

	return views
}

func (s *Service) roomStatus(ctx context.Context, room config.RoomConfig) RoomView {
	view := RoomView{
		Key:    room.Key,
		Name:   room.Name,
		RoomID: room.RoomID,
	}

	session, ok := s.sessions[room.Key]
	if !ok {
		view.Err = fmt.Errorf("session for room %q not found", room.Key)
		return view
	}

	status, err := session.status(ctx)
	if err != nil {
		view.Err = err
		return view
	}

	byNumber := make(map[int]screenmate.AirConditioner, len(status.AirConditioners))
	for _, ac := range status.AirConditioners {
		byNumber[ac.Number] = ac
	}

	numbers := make([]int, 0, len(room.Conditioners))
	for number := range room.Conditioners {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)

	for _, number := range numbers {
		ac, found := byNumber[number]

		view.Conditioners = append(view.Conditioners, ConditionerView{
			RoomKey: room.Key,
			Number:  number,
			Comment: room.Conditioners[number],
			Power:   ac.Power,
			Found:   found,
		})
	}

	return view
}

func (s *Service) TogglePower(ctx context.Context, roomKey string, acNumber int) error {
	session, ok := s.sessions[roomKey]
	if !ok {
		return fmt.Errorf("room %q not found", roomKey)
	}

	if _, ok := session.room.Conditioners[acNumber]; !ok {
		return fmt.Errorf("air conditioner %d is not configured for room %q", acNumber, roomKey)
	}

	return session.togglePower(ctx, acNumber)
}

func (s *roomSession) getClient() (*screenmate.Client, error) {
	if s.client != nil {
		return s.client, nil
	}

	client, err := screenmate.NewClient(
		s.baseURL,
		s.room.Username,
		s.room.Password,
		s.room.RoomID,
	)
	if err != nil {
		return nil, err
	}

	s.client = client
	return client, nil
}

func (s *roomSession) maybeResetIdle() {
	if s.client == nil || s.lastUsed.IsZero() {
		return
	}

	if time.Since(s.lastUsed) > 30*time.Minute {
		s.client.Reset()
	}
}

func (s *roomSession) status(ctx context.Context) (screenmate.RoomStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maybeResetIdle()

	client, err := s.getClient()
	if err != nil {
		return screenmate.RoomStatus{}, err
	}

	status, err := client.Status(ctx)
	if err != nil {
		client.Reset()
		return screenmate.RoomStatus{}, err
	}

	s.lastUsed = time.Now()

	return status, nil
}

func (s *roomSession) togglePower(ctx context.Context, acNumber int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maybeResetIdle()

	client, err := s.getClient()
	if err != nil {
		return err
	}

	if err := client.TogglePower(ctx, acNumber); err != nil {
		client.Reset()
		return err
	}

	s.lastUsed = time.Now()

	return nil
}

func (s *roomSession) togglePowerIfState(
	ctx context.Context,
	acNumber int,
	expectedPower bool,
) (ToggleResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maybeResetIdle()

	client, err := s.getClient()
	if err != nil {
		return ToggleResult{}, err
	}

	result, err := client.TogglePowerIfState(ctx, acNumber, expectedPower)
	if err != nil {
		client.Reset()
		return ToggleResult{}, err
	}

	s.lastUsed = time.Now()

	return ToggleResult{
		Toggled:      result.Toggled,
		StateChanged: result.StateChanged,
		CurrentPower: result.CurrentPower,
	}, nil
}

func (s *Service) TogglePowerIfState(
	ctx context.Context,
	roomKey string,
	acNumber int,
	expectedPower bool,
) (ToggleResult, error) {
	session, ok := s.sessions[roomKey]
	if !ok {
		return ToggleResult{}, fmt.Errorf("room %q not found", roomKey)
	}

	if _, ok := session.room.Conditioners[acNumber]; !ok {
		return ToggleResult{}, fmt.Errorf("air conditioner %d is not configured for room %q", acNumber, roomKey)
	}

	return session.togglePowerIfState(ctx, acNumber, expectedPower)
}
