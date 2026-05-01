# 🔥 Locolive – Production Privacy + Block + Chat System

## 📌 Overview

Ye system ensure karta hai ki:

* Users ke beech **controlled visibility** ho
* Block hone par **full isolation** ho
* Chat system **real-time secure** ho
* Har request **central rule engine** se validate ho

---

# 🧠 Core Rule Engine (MOST IMPORTANT)

```go
func CanUserAccess(viewerID, targetID uuid.UUID) (bool, string) {

    if IsBlocked(viewerID, targetID) {
        return false, "BLOCKED"
    }

    state := getPrivacyState(targetID)

    if state.PanicMode {
        return false, "PANIC"
    }

    if state.GhostMode {
        return false, "GHOST"
    }

    if state.IsPrivate && !isFollower(viewerID, targetID) {
        return false, "PRIVATE"
    }

    return true, "ALLOWED"
}
```

---

# 🔥 Rule Priority

```text
Block > Panic > Ghost > Private > Public
```

---

# 🗄️ Database Design

## Users Table

```sql
ALTER TABLE users ADD COLUMN:
- is_private BOOLEAN DEFAULT false,
- ghost_mode BOOLEAN DEFAULT false,
- panic_mode BOOLEAN DEFAULT false;
```

## Blocks Table

```sql
CREATE TABLE blocks (
  blocker_id UUID,
  blocked_id UUID,
  created_at TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id)
);
```

---

# ⚙️ Backend Flow (Go)

---

## 1️⃣ Block User

```go
func BlockUser(c *gin.Context) {

    blockerID := getUserID(c)
    blockedID := parseID(c.Param("id"))

    if blockerID == blockedID {
        c.JSON(400, gin.H{"error": "cannot block yourself"})
        return
    }

    insertBlock(blockerID, blockedID)

    removeFollow(blockerID, blockedID)
    deleteFollowRequests(blockerID, blockedID)

    closeChat(blockerID, blockedID)

    ws.DisconnectUser(blockedID)

    cache.Delete("privacy:" + blockedID.String())

    logAction("USER_BLOCKED", blockerID, blockedID)

    c.JSON(200, gin.H{"message": "blocked"})
}
```

---

## 2️⃣ Unblock User

```go
func UnblockUser(c *gin.Context) {

    blockerID := getUserID(c)
    blockedID := parseID(c.Param("id"))

    deleteBlock(blockerID, blockedID)

    cache.Delete("privacy:" + blockedID.String())

    c.JSON(200, gin.H{"message": "unblocked"})
}
```

---

## 3️⃣ Middleware (Global Protection)

```go
func PrivacyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {

        viewerID := getUserID(c)
        targetID := parseID(c.Param("id"))

        allowed, reason := CanUserAccess(viewerID, targetID)

        if !allowed {
            if reason == "PRIVATE" {
                c.JSON(403, gin.H{"error": "private account"})
            } else {
                c.JSON(404, gin.H{"error": "not found"})
            }
            c.Abort()
            return
        }

        c.Next()
    }
}
```

---

# 💬 Chat Integration

## Message Send Check

```go
func SendMessage(senderID, receiverID uuid.UUID) error {

    allowed, _ := CanUserAccess(senderID, receiverID)

    if !allowed {
        return errors.New("chat not allowed")
    }

    return saveMessage()
}
```

---

## WebSocket Connection

```go
if !CanUserAccess(userID, targetID) {
    conn.WriteJSON({
        "type": "error",
        "message": "access denied"
    })
    return
}
```

---

## On Block Event

```go
func OnBlock(blocker, blocked uuid.UUID) {

    closeChatRoom(blocker, blocked)

    ws.DisconnectUser(blocked)

    ws.Send(blocked, {
        "type": "blocked",
        "by": blocker
    })
}
```

---

# ⚡ Redis Cache

```text
privacy:state:{userID}
block:{userA}:{userB}
```

## Invalidate When:

* block / unblock
* privacy change
* follow change

---

# 🎨 Frontend Flow (React)

---

## 1️⃣ Block Button

```tsx
const handleBlock = async () => {
  await api.post(`/users/block/${userId}`);
  navigate("/home");
};
```

---

## 2️⃣ Profile Page

```tsx
if (error?.response?.status === 404) {
  return <div>User not found</div>;
}
```

---

## 3️⃣ Chat Page

```tsx
if (error?.response?.status === 403 || 404) {
  return <div>Chat not available</div>;
}
```

---

## 4️⃣ WebSocket Listener

```tsx
socket.on("blocked", () => {
  alert("You are blocked");
  navigate("/home");
});
```

---

# 🔄 Complete Flow

```text
User clicks Block →
DB insert →
Follow removed →
Chat closed →
WebSocket disconnect →
Cache invalidated →
UI update →
User disappears everywhere
```

---

# 🔓 Unblock Flow

```text
User clicks Unblock →
DB delete →
Cache clear →
Visibility restored
```

---

# 🔐 Security Rules

* JWT required
* Prevent self-block
* Rate limiting
* Input validation
* Audit logging
* Cache invalidation

---

# ⚠️ Edge Cases

* Already blocked → ignore
* Both users blocked → still blocked
* Chat active → force close
* Search cache → refresh

---

# 🚀 Production Checklist

* [ ] Block system working
* [ ] Privacy engine centralized
* [ ] Chat checks applied
* [ ] WebSocket sync working
* [ ] Redis cache working
* [ ] Middleware applied everywhere
* [ ] UI handles 403/404
* [ ] Audit logs enabled

---

# 🧠 Final Summary

> Block + Privacy + Chat = **Single unified system**

Agar ye properly implement nahi hua:

❌ Data leak
❌ Users bypass
❌ Chat exploit

---

# 🎯 Golden Rule

```text
Always check CanUserAccess BEFORE any action
```

---
