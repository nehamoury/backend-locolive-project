# 🔐 Username Uniqueness System (Locolive Style) — Go Backend

## 📌 Overview

This document defines a robust system to ensure globally unique usernames in an application like Locolive.

The system ensures:

* Strict uniqueness
* Case-insensitive validation
* High performance under concurrency
* Protection against abuse and race conditions

---

## 🧱 Tech Stack

* Language: Go (Golang)
* Framework: net/http / Gin (optional)
* Database: PostgreSQL / MySQL
* Cache (optional): Redis

---

## 📏 Username Rules

* Must be unique
* Case-insensitive (Rahul == rahul)
* Allowed: `a-z`, `0-9`, `_`
* Length: 3–20 characters
* Must start with a letter

---

## 🧹 Normalization Strategy

Before storing:

* Convert to lowercase
* Trim spaces
* Remove invalid characters

### Example

Input: `Rahul_123`
Stored: `rahul_123`

---

## 🗄️ Database Schema

### PostgreSQL Example

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(20) NOT NULL UNIQUE,
    username_original VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Case-insensitive index (important)
CREATE UNIQUE INDEX unique_username_lower
ON users (LOWER(username));
```

---

## ⚙️ Backend Implementation (Go)

### 1. Username Validation Function

```go
package utils

import (
	"regexp"
	"strings"
)

var usernameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{2,19}$`)

func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func IsValidUsername(username string) bool {
	return usernameRegex.MatchString(username)
}
```

---

### 2. Check Availability

```go
func IsUsernameAvailable(db *sql.DB, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	err := db.QueryRow(query, username).Scan(&exists)
	return !exists, err
}
```

---

### 3. Create User (Safe Insert)

```go
func CreateUser(db *sql.DB, username string) error {
	query := `INSERT INTO users (username) VALUES ($1)`
	_, err := db.Exec(query, username)
	return err
}
```

⚠️ IMPORTANT:

* Always rely on DB UNIQUE constraint
* Handle duplicate error (`unique_violation`)

---

### 4. HTTP Handler (Gin Example)

```go
func Register(c *gin.Context) {
	username := utils.NormalizeUsername(c.PostForm("username"))

	if !utils.IsValidUsername(username) {
		c.JSON(400, gin.H{"error": "Invalid username"})
		return
	}

	err := CreateUser(db, username)
	if err != nil {
		c.JSON(400, gin.H{"error": "Username already taken"})
		return
	}

	c.JSON(200, gin.H{"message": "User created"})
}
```

---

## 🔄 Race Condition Handling

* DO NOT rely only on "check then insert"
* Always:

  1. Try insert
  2. Catch duplicate error

---

## ⚡ Optional: Username Reservation (Redis)

### Flow:

1. User enters username
2. Reserve in Redis for 5 minutes
3. Prevent others from using it temporarily

```bash
SET username:rahul123 reserved EX 300 NX
```

---

## 🔍 Username Suggestions Logic

### Strategy:

* Append numbers → rahul123
* Add prefix → real_rahul
* Add suffix → rahul_live

### Example Code

```go
func SuggestUsernames(base string) []string {
	return []string{
		base + "123",
		"real_" + base,
		base + "_live",
	}
}
```

---

## 🚫 Blocked Usernames

Maintain a blacklist:

```go
var blocked = map[string]bool{
	"admin": true,
	"support": true,
	"official": true,
}
```

---

## 🔐 Security Measures

* Rate limiting (per IP)
* CAPTCHA (optional)
* Prevent brute-force username attempts

---

## 🧾 Username History (Optional)

```sql
CREATE TABLE username_history (
    id SERIAL PRIMARY KEY,
    user_id INT,
    old_username VARCHAR(20),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 🔗 Profile URL

```
https://yourapp.com/{username}
```

---

## 🧪 Testing Cases

| Input    | Expected Output               |
| -------- | ----------------------------- |
| Rahul123 | ❌ (if exists)                 |
| rahul_01 | ✅                             |
| RAJ      | ❌ (too short or invalid rule) |
| user@123 | ❌                             |

---

## 🚀 Production Checklist

* [x] DB UNIQUE constraint
* [x] Case-insensitive handling
* [x] Validation regex
* [x] Error handling
* [x] Rate limiting
* [x] Suggestions system
* [x] Logging & monitoring

---

## 🎯 Conclusion

A strong username system must enforce uniqueness at:

1. Application level
2. Database level

And must handle:

* Concurrency
* Abuse
* User experience

This ensures scalability and reliability in real-world apps.

---
