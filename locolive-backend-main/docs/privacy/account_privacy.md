# Account Privacy System (Enhanced)

## Overview
Controls visibility of user content based on privacy settings.

---

## Privacy Scope
This system controls access to:
- Profile
- Posts
- Stories
- Followers/Following list

---

## Rules
- Public → anyone can view
- Private → only approved followers
- Owner → always full access

---

## Database
```sql
ALTER TABLE users 
ADD COLUMN is_private BOOLEAN DEFAULT false,
ADD COLUMN privacy_updated_at TIMESTAMP;
```

---

## API

### Update Privacy
PATCH /user/privacy

Request:
```json
{
  "is_private": true
}
```

Response:
```json
{
  "message": "Privacy updated"
}
```

---

## Core Logic (Go)

```go
func CanViewProfile(viewerID, ownerID int) bool {

    // Self access
    if viewerID == ownerID {
        return true
    }

    // Block check
    if isBlocked(ownerID, viewerID) {
        return false
    }

    // Private account check
    if isPrivate(ownerID) && !isFollower(viewerID, ownerID) {
        return false
    }

    return true
}
```

---

## Content-Level Access

```go
func CanViewContent(viewerID, ownerID int) bool {
    return CanViewProfile(viewerID, ownerID)
}
```

---

## Follow Request Enforcement

```go
if isPrivate(ownerID) {
    createFollowRequest()
} else {
    createFollower()
}
```

---

## Error Handling

- 403 Forbidden → Private account
- 404 Not Found → If blocked (hide existence)

---

## Performance Optimization

- Cache follower relationships (Redis)
- Cache privacy flag

---

## Audit Logging

```sql
privacy_logs (
  user_id,
  old_value,
  new_value,
  changed_at
)
```

---

## Edge Cases

- Blocked users → always denied
- Self-view → always allowed
- Deactivated accounts → no access
- Suspended users → restricted access

---

## Security Considerations

- Never trust frontend
- Always validate on backend
- Prevent IDOR attacks

---

## Future Enhancements

- Per-content privacy (posts vs stories)
- Close friends override
- Privacy tiers (followers / mutuals / custom)

---

## Testing Cases

| Scenario | Expected |
|--------|---------|
| Private + not follower | ❌ Denied |
| Private + follower | ✅ Allowed |
| Blocked | ❌ Denied |
| Self | ✅ Allowed |

---

## Final Note
Privacy must be enforced at every API layer, not just UI.