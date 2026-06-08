package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"screenmate-bot/internal/config"
	"screenmate-bot/internal/screenmate"
)

type Service struct {
	baseURL     string
	rooms       []config.RoomConfig
	maxParallel int
}

func New(cfg config.Config) *Service {
	maxParallel := cfg.ScreenMate.MaxParallelRooms
	if maxParallel <= 0 {
		maxParallel = 3
	}

	return &Service{
		baseURL:     cfg.ScreenMate.BaseURL,
		rooms:       cfg.Rooms,
		maxParallel: maxParallel,
	}
}

func (s *Service) AllRoomsStatus(ctx context.Context) []RoomView {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

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

	client, err := screenmate.NewClient(
		s.baseURL,
		room.Username,
		room.Password,
		room.RoomID,
	)
	if err != nil {
		view.Err = err
		return view
	}

	status, err := client.Status(ctx)
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
	var room config.RoomConfig
	found := false

	for _, r := range s.rooms {
		if r.Key == roomKey {
			room = r
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("room %q not found", roomKey)
	}

	if _, ok := room.Conditioners[acNumber]; !ok {
		return fmt.Errorf("air conditioner %d is not configured for room %q", acNumber, roomKey)
	}

	client, err := screenmate.NewClient(
		s.baseURL,
		room.Username,
		room.Password,
		room.RoomID,
	)
	if err != nil {
		return err
	}

	return client.TogglePower(ctx, acNumber)
}
