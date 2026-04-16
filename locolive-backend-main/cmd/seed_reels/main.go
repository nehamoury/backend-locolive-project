package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"privacy-social-backend/internal/config"
	"privacy-social-backend/internal/repository/db"

	_ "github.com/lib/pq"
)

func main() {
	config, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	queries := db.New(conn)
	ctx := context.Background()

	// Find the test user
	user, err := queries.GetUserByUsername(ctx, "testmap")
	if err != nil {
		// Fallback to any user if testmap not found
		rows, err := conn.QueryContext(ctx, "SELECT id FROM users LIMIT 1")
		if err != nil {
			log.Fatal("no users found in db")
		}
		defer rows.Close()
		if rows.Next() {
			err = rows.Scan(&user.ID)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("no users in database")
		}
	}

	fmt.Printf("Seeding for user: %s (%s)\n", user.Username, user.ID)

	reels := []struct {
		url      string
		caption  string
		lat, lng float64
		ai       bool
	}{
		{"https://assets.mixkit.co/videos/preview/mixkit-girl-in-neon-lights-12853-large.mp4", "Vibrant colors in the night! 🌈✨", 21.2118, 81.3164, false},
		{"https://assets.mixkit.co/videos/preview/mixkit-man-dancing-under-the-rain-4458-large.mp4", "Dance like no one is watching 🕺💦 #RainyVibes", 21.2218, 81.3264, true},
		{"https://assets.mixkit.co/videos/preview/mixkit-tree-with-yellow-leaves-low-angle-view-1888-large.mp4", "Nature's gold 🍁🌻 #Autumn", 21.2018, 81.3064, false},
	}

	for _, r := range reels {
		// PostGIS geom calculation
		geom := fmt.Sprintf("SRID=4326;POINT(%f %f)", r.lng, r.lat)

		_, err := queries.CreateReel(ctx, db.CreateReelParams{
			UserID:        user.ID,
			VideoUrl:      r.url,
			Caption:       sql.NullString{String: r.caption, Valid: true},
			IsAiGenerated: r.ai,
			LocationName:  sql.NullString{String: "Durg, Chhattisgarh", Valid: true},
			Geom:          geom,
		})
		if err != nil {
			fmt.Printf("Failed to seed reel: %v\n", err)
		} else {
			fmt.Printf("Seeded reel: %s\n", r.caption)
		}
	}

	fmt.Println("Seeding complete!")
}
