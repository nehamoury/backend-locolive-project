package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"privacy-social-backend/internal/repository/db"
)

func main() {
	_ = godotenv.Load("app.env")
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_SOURCE")
	}
	if dbURL == "" {
		log.Fatal("DB_URL or DB_SOURCE is not set in app.env")
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	queries := db.New(conn)
	ctx := context.Background()

	categories := []db.CreateCategoryParams{
		{Name: "Food", Slug: "food", Icon: sql.NullString{String: "restaurant", Valid: true}, Color: sql.NullString{String: "#FF6B6B", Valid: true}, IsActive: true},
		{Name: "Travel", Slug: "travel", Icon: sql.NullString{String: "flight", Valid: true}, Color: sql.NullString{String: "#4D96FF", Valid: true}, IsActive: true},
		{Name: "Fashion", Slug: "fashion", Icon: sql.NullString{String: "checkroom", Valid: true}, Color: sql.NullString{String: "#FF8E53", Valid: true}, IsActive: true},
		{Name: "Sports", Slug: "sports", Icon: sql.NullString{String: "sports-basketball", Valid: true}, Color: sql.NullString{String: "#38E54D", Valid: true}, IsActive: true},
		{Name: "Music", Slug: "music", Icon: sql.NullString{String: "music-note", Valid: true}, Color: sql.NullString{String: "#9C27B0", Valid: true}, IsActive: true},
		{Name: "Events", Slug: "events", Icon: sql.NullString{String: "event", Valid: true}, Color: sql.NullString{String: "#F4D160", Valid: true}, IsActive: true},
		{Name: "Business", Slug: "business", Icon: sql.NullString{String: "business", Valid: true}, Color: sql.NullString{String: "#607D8B", Valid: true}, IsActive: true},
		{Name: "Education", Slug: "education", Icon: sql.NullString{String: "school", Valid: true}, Color: sql.NullString{String: "#00BCD4", Valid: true}, IsActive: true},
		{Name: "Shopping", Slug: "shopping", Icon: sql.NullString{String: "shopping-bag", Valid: true}, Color: sql.NullString{String: "#E91E63", Valid: true}, IsActive: true},
		{Name: "News", Slug: "news", Icon: sql.NullString{String: "article", Valid: true}, Color: sql.NullString{String: "#3F51B5", Valid: true}, IsActive: true},
		{Name: "Jobs", Slug: "jobs", Icon: sql.NullString{String: "work", Valid: true}, Color: sql.NullString{String: "#FF9800", Valid: true}, IsActive: true},
		{Name: "Technology", Slug: "technology", Icon: sql.NullString{String: "computer", Valid: true}, Color: sql.NullString{String: "#009688", Valid: true}, IsActive: true},
		{Name: "Gaming", Slug: "gaming", Icon: sql.NullString{String: "sports-esports", Valid: true}, Color: sql.NullString{String: "#673AB7", Valid: true}, IsActive: true},
		{Name: "Health", Slug: "health", Icon: sql.NullString{String: "favorite", Valid: true}, Color: sql.NullString{String: "#F44336", Valid: true}, IsActive: true},
		{Name: "Local", Slug: "local", Icon: sql.NullString{String: "location-on", Valid: true}, Color: sql.NullString{String: "#795548", Valid: true}, IsActive: true},
	}

	log.Println("Seeding categories...")
	for _, cat := range categories {
		_, err := queries.GetCategoryBySlug(ctx, cat.Slug)
		if err == sql.ErrNoRows {
			_, err = queries.CreateCategory(ctx, cat)
			if err != nil {
				log.Printf("Failed to create category %s: %v", cat.Name, err)
			} else {
				log.Printf("Created category: %s", cat.Name)
			}
		} else if err != nil {
			log.Printf("Error checking category %s: %v", cat.Name, err)
		} else {
			log.Printf("Category %s already exists. Skipping.", cat.Name)
		}
	}
	log.Println("Seed complete.")
}
