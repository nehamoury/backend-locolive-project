# 🚀 Locolive — Production-Ready Privacy & Security System

## 📌 Overview

Locolive is a **presence-based privacy social platform** where user visibility depends on activity.

Core principle:

> “User exists only when active.”

This document defines a **unified Privacy + Security + Visibility system** for production use.

---

# 🧠 1. Core Architecture

## Single Source of Truth

All access decisions must go through ONE function:

```go
func CanUserAccess(viewerID, targetID int, resource string) bool
```

---

# ⚖️ 2. Priority Rule Engine (CRITICAL)

Rules must be evaluated in this order:

1. 🚫 **Block System (Highest Priority)**
2. 🚨 **Panic Mode**
3. 👻 **Ghost Mode**
4. 🔒 **Private Account**
5. ⏳ **Visibility Engine (last_active_at)**
6. 🌐 Default (Public)

---

# 🔐 3. Privacy Components

## 3.1 Account Privacy

* Public → anyone can view
* Private → only followers

DB:

```sql
ALTER TABLE users ADD COLUMN is_private BOOLEAN DEFAULT false;
```

---

## 3.2 Ghost Mode

* User becomes instantly invisible

```sql
ALTER TABLE users ADD COLUMN ghost_mode BOOLEAN DEFAULT false;
```

---

## 3.3 Panic Mode

* Emergency full hide/delete

```sql
ALTER TABLE users ADD COLUMN panic_mode BOOLEAN DEFAULT false;
```

---

## 3.4 Close Friends (Optional Override)

* Story-level visibility

---

# 🚫 4. Blocking System

## Rules

* No interaction
* No visibility
* Remove relationships

DB:

```sql
CREATE TABLE blocks (
  blocker_id INT,
  blocked_id INT,
  created_at TIMESTAMP,
  UNIQUE(blocker_id, blocked_id)
);
```

---

# ⏳ 5. Visibility Engine (CORE FEATURE)

## Based on `last_active_at`

| State  | Condition | Visibility    |
| ------ | --------- | ------------- |
| Active | < 24h     | Fully visible |
| Fading | < 3 days  | Limited       |
| Hidden | > 3 days  | Invisible     |

DB:

```sql
ALTER TABLE users ADD COLUMN last_active_at TIMESTAMP;
```

---

# 🔍 6. Unified Access Logic (Go)

```go
func CanUserAccess(viewerID, targetID int) bool {

    // Self access
    if viewerID == targetID {
        return true
    }

    // 1. Block check
    if isBlocked(viewerID, targetID) {
        return false
    }

    user := getUser(targetID)

    // 2. Panic mode
    if user.PanicMode {
        return false
    }

    // 3. Ghost mode
    if user.GhostMode {
        return false
    }

    // 4. Private account
    if user.IsPrivate && !isFollower(viewerID, targetID) {
        return false
    }

    // 5. Visibility engine
    if isHidden(user.LastActiveAt) {
        return false
    }

    return true
}
```

---

# 🔄 7. Follow System

## Logic

```go
if target.IsPrivate {
    createFollowRequest()
} else {
    createFollower()
}
```

---

# 💬 8. Messaging Security

* Blocked → ❌ no message
* Non-followers → configurable

---

# 🔐 9. Authentication & Security

## Auth

* JWT (Access + Refresh)

## Password

* bcrypt hashing

## Sessions

```sql
sessions (
  user_id,
  token,
  device,
  expires_at
)
```

---

# ⚡ 10. Rate Limiting

* Per IP
* Per user

Use:

* Redis
* Token bucket

---

# 🛡️ 11. Abuse Protection

* Username blacklist
* Spam detection
* Report system

---

# 🧾 12. Audit Logs

```sql
user_activity_logs (
  user_id,
  action,
  created_at
)
```

---

# 🚀 13. Performance Optimization

* Redis caching:

  * followers
  * blocks
  * visibility

* DB Indexing:

```sql
CREATE INDEX idx_users_active ON users(last_active_at);
```

---

# 🔒 14. API Security Rules

Every API MUST:

* Validate user identity
* Call `CanUserAccess`
* Prevent IDOR attacks

---

# ⚠️ 15. Error Handling Strategy

| Case         | Response             |
| ------------ | -------------------- |
| Blocked      | 404 (hide existence) |
| Private      | 403                  |
| Unauthorized | 401                  |

---

# 🧪 16. Testing Scenarios

| Scenario               | Expected    |
| ---------------------- | ----------- |
| Blocked user           | ❌ no access |
| Ghost mode             | ❌ invisible |
| Private + not follower | ❌ denied    |
| Active user            | ✅ allowed   |

---

# 🐳 17. Deployment Ready

* Dockerized services
* Nginx reverse proxy
* HTTPS enabled
* Env-based config

---

# 🎯 18. Final Principles

* Never trust frontend
* Always validate on backend
* Use single rule engine
* Security > Features

---

# 🏁 Conclusion

This system ensures:

* 🔐 Strong privacy control
* ⚡ High performance
* 🛡️ Abuse resistance
* 📈 Scalability

It is production-ready and suitable for real-world deployment.
# 🔥 Production Stability Layer

## Middleware
- Auth middleware (JWT validation)
- Permission middleware (CanUserAccess)

## Failure Handling
- Retry DB queries (max 3)
- Redis fallback to DB
- Timeout handling

## Data Safety
- Use soft delete instead of hard delete
- Add `deleted_at` column

## Consistency
- Use DB transactions
- Prevent duplicate actions

## Monitoring
- Log all critical actions
- Track errors
- Setup alerts