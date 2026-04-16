package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"

	"privacy-social-backend/internal/config"
	"privacy-social-backend/internal/repository"
	"privacy-social-backend/internal/repository/db"
	"privacy-social-backend/internal/service/location"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	conn, err := sql.Open(cfg.DBDriver, cfg.DBSource)
	if err != nil {
		log.Fatalf("cannot connect to db: %v", err)
	}
	defer conn.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddress,
	})

	store := repository.NewStore(conn)
	// Hub is optional for simulation if we don't care about the WS alert right now
	locService := location.NewRedisLocationService(rdb, store, nil)

	ctx := context.Background()

	// 1. Get or Create User 1 (The current user 'nemo' or similar)
	// For simulation, we'll just use the first two users found or create demo ones
	users, err := store.ListUsers(ctx, db.ListUsersParams{Limit: 2, Offset: 0})
	if err != nil || len(users) < 1 {
		log.Fatalf("Please register at least one user first: %v", err)
	}

	u1 := users[0]
	var u2ID uuid.UUID

	if len(users) < 2 {
		fmt.Println("Creating a second user for simulation...")
		// Create a dummy second user
		// Note: This requires all fields to be handled. Simplified for placeholder.
		// In a real app, you'd have a 'test' user.
		log.Fatal("Please register a second user in the app first to simulate a crossing.")
	} else {
		u2ID = users[1].ID
	}

	fmt.Printf("Simulating crossing between %s and %s...\n", u1.Username, users[1].Username)

	// Coordinates for "Magnet Mall" (Demo location)
	lat, lng := 21.2467, 81.6337

	// Ping from User 2 first
	fmt.Println("Ping from User 2...")
	err = locService.UpdateUserLocation(ctx, u2ID, lat, lng)
	if err != nil {
		log.Fatalf("User 2 ping failed: %v", err)
	}

	// Ping from User 1 (should trigger crossing)
	fmt.Println("Ping from User 1...")
	err = locService.UpdateUserLocation(ctx, u1.ID, lat, lng)
	if err != nil {
		log.Fatalf("User 1 ping failed: %v", err)
	}

	fmt.Println("✅ Crossing simulated successfully!")
	fmt.Println("Refresh your Crossings page in the app to see the result.")
}
