# Admin API Guide

## Authentication

Admin endpoints require:
1. Valid JWT token (same as regular auth)
2. User role must be `admin` or `moderator`

**Setting Admin Role:**
```sql
-- Update user role in database
UPDATE users SET role = 'admin' WHERE phone = '+1234567890';
```

## Admin Endpoints

Base path: `/admin/*`

### 1. User Management

#### List All Users (Paginated)
```bash
curl -X GET "http://localhost:8080/admin/users?page=1&page_size=20" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

**Response:**
```json
{
  "users": [...],
  "total": 150,
  "page": 1
}
```

#### Ban/Unban User
```bash
curl -X POST http://localhost:8080/admin/users/ban \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN" \
  -d '{
    "user_id": "USER_UUID",
    "ban": true
  }'
```

#### Delete User
```bash
curl -X DELETE http://localhost:8080/admin/users/USER_UUID \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

### 2. Analytics & Statistics

#### Get System Stats
```bash
curl -X GET http://localhost:8080/admin/stats \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

**Response:**
```json
{
  "totalUsers": 1500,
  "newUsers24h": 45,
  "activeUsers": 120,
  "totalConnections": 500,
  "reelsToday": 30,
  "crossingsToday": 15
}
```

### 3. Content Moderation

#### List Reports
```bash
curl -X GET "http://localhost:8080/admin/reports?resolved=false&page=1&page_size=20" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

#### Resolve Report
```bash
curl -X PUT http://localhost:8080/admin/reports/REPORT_UUID/resolve \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

#### List All Stories
```bash
curl -X GET "http://localhost:8080/admin/stories?page=1&page_size=50" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

#### Delete Story
```bash
curl -X DELETE http://localhost:8080/admin/stories/STORY_UUID \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

### 4. Advanced Moderation & Logs

#### List Activity Logs
```bash
curl -X GET "http://localhost:8080/admin/activity/logs?page=1&page_size=20" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

#### Moderate Comments
```bash
# List all comments
curl -X GET "http://localhost:8080/admin/comments?page=1&page_size=20"

# Take action
curl -X POST http://localhost:8080/admin/comments/moderate \
  -d '{"comment_id": "UUID", "source": "post", "action": "delete"}'
```

### 5. System Tools

#### Broadcast Notification
```bash
curl -X POST http://localhost:8080/admin/notifications/send \
  -d '{"title": "Update!", "message": "New features added", "target": "all"}'
```

#### View Real-time Crossing Logs
```bash
curl -X GET http://localhost:8080/admin/crossings?page=1&page_size=20
```

## Admin Workflow Examples

### 1. Moderate Reported Content
```bash
# 1. Get unresolved reports
curl -X GET "http://localhost:8080/admin/reports?resolved=false&page=1&page_size=10" \
  -H "Authorization: Bearer ADMIN_TOKEN"

# 2. Review and delete offending story
curl -X DELETE http://localhost:8080/admin/stories/STORY_UUID \
  -H "Authorization: Bearer ADMIN_TOKEN"

# 3. Ban the user
curl -X POST http://localhost:8080/admin/users/ban \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -d '{"user_id": "USER_UUID", "ban": true}'

# 4. Mark report as resolved
curl -X PUT http://localhost:8080/admin/reports/REPORT_UUID/resolve \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

### 2. Monitor System Health
```bash
# Get real-time statistics
curl -X GET http://localhost:8080/admin/stats \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

## Security Features

✅ **Role-Based Access Control**: Only admin/moderator roles
✅ **Double Authentication**: JWT + Role verification
✅ **Audit Trail**: All admin actions logged
✅ **Pagination**: Prevents data overload
✅ **Soft Delete Ready**: Can implement soft deletes

## Admin Capabilities

- **User Moderation**: Ban, unban, delete users
- **Content Control**: Delete stories, resolve reports
- **Analytics**: Real-time user and content statistics
- **Bulk Operations**: Paginated list views
- **Safety Tools**: Shadow ban, trust level management
