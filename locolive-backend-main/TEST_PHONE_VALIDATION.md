# 🧪 Phone Validation Test Guide

## ✅ TEST 1: Valid Phone Numbers (Should Pass)

### Indian number without +
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "9876543210",
    "email": "realuser@gmail.com",
    "username": "testuser123",
    "full_name": "Test User",
    "password": "password123"
  }'
```

### Indian number with +
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+919876543210",
    "email": "realuser2@gmail.com",
    "username": "testuser456",
    "full_name": "Test User 2",
    "password": "password123"
  }'
```

**Expected:** `201 Created` + tokens returned (dev mode: OTP printed in console)

---

## ❌ TEST 2: Invalid Formats (Should Fail)

### Too short
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "12345",
    "email": "user1@test.com",
    "username": "shortuser",
    "full_name": "Short User",
    "password": "password123"
  }'
```
**Expected:** `400 Bad Request` + `"Invalid phone number format"`

### Letters in phone
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "abcdefghij",
    "email": "user2@test.com",
    "username": "letteruser",
    "full_name": "Letter User",
    "password": "password123"
  }'
```
**Expected:** `400 Bad Request` + `"Invalid phone number format"`

### Empty phone
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "",
    "email": "user3@test.com",
    "username": "emptyuser",
    "full_name": "Empty User",
    "password": "password123"
  }'
```
**Expected:** `400 Bad Request` (binding validation fails)

### Missing + with invalid digits
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+12345678901234567",
    "email": "user4@test.com",
    "username": "toolonguser",
    "full_name": "Too Long User",
    "password": "password123"
  }'
```
**Expected:** `400 Bad Request` + `"Invalid phone number format"` (too many digits)

---

## 🔄 TEST 3: Phone Number Normalization

### Input with spaces/dashes (should auto-clean)
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "98765-43210",
    "email": "normalize@test.com",
    "username": "normuser",
    "full_name": "Normalize User",
    "password": "password123"
  }'
```
**Expected:** `201 Created` (normalized to `+919876543210`)

### Leading zero (India format)
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "09876543210",
    "email": "leadingzero@test.com",
    "username": "zerouser",
    "full_name": "Zero User",
    "password": "password123"
  }'
```
**Expected:** `201 Created` (normalized to `+919876543210`, leading 0 removed)

---

## 🚫 TEST 4: VoIP Blocking (Production Only)

> ⚠️ Ye sirf tab test hoga jab `TWILIO_ACCOUNT_SID` set ho aur `ENVIRONMENT=production`

```bash
# Twilio VoIP number (known test number)
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+15005550006",
    "email": "voipuser@test.com",
    "username": "voipuser",
    "full_name": "VoIP User",
    "password": "password123"
  }'
```
**Expected (if Twilio configured):** `400 Bad Request` + `"virtual/VoIP phone numbers are not allowed"`

---

## 📱 TEST 5: Complete Profile (Google OAuth users)

Pehle Google se login karo ya direct token use karo:

```bash
curl -X POST http://localhost:8080/api/users/complete-profile \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "username": "googleuser123",
    "phone": "9876543210"
  }'
```

**Expected:** `200 OK` + tokens (phone normalized + validated)

---

## 🔍 TEST 6: Check OTP in Dev Mode

Jab Twilio configured nahi hai, OTP console mein print hota hai:

```bash
# Signup ke baad terminal mein dekho:
------------
[DEVELOPMENT MODE - SMS LOG]
To: +919876543210
Message: Your Locolive verification code is: 123456
------------
```

Us OTP se verify karo:

```bash
curl -X POST http://localhost:8080/api/auth/verify-phone \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "code": "123456"
  }'
```

---

## 🧪 Quick Test Script

```powershell
# Run all tests at once
Write-Host "`n=== TEST 1: Valid Indian Number ===" -ForegroundColor Green
curl.exe -s -X POST http://localhost:8080/api/users `
  -H "Content-Type: application/json" `
  -d '{"phone":"9876543210","email":"valid@gmail.com","username":"validuser","full_name":"Valid","password":"test123"}' | ConvertFrom-Json | Format-List

Write-Host "`n=== TEST 2: Invalid Format ===" -ForegroundColor Red
curl.exe -s -X POST http://localhost:8080/api/users `
  -H "Content-Type: application/json" `
  -d '{"phone":"12345","email":"invalid@gmail.com","username":"invuser","full_name":"Invalid","password":"test123"}' | ConvertFrom-Json | Format-List

Write-Host "`n=== TEST 3: Normalization (spaces) ===" -ForegroundColor Yellow
curl.exe -s -X POST http://localhost:8080/api/users `
  -H "Content-Type: application/json" `
  -d '{"phone":"98765-43210","email":"norm@gmail.com","username":"normuser","full_name":"Normalize","password":"test123"}' | ConvertFrom-Json | Format-List

Write-Host "`n=== TEST 4: Letters in Phone ===" -ForegroundColor Red
curl.exe -s -X POST http://localhost:8080/api/users `
  -H "Content-Type: application/json" `
  -d '{"phone":"abcdefghij","email":"letters@gmail.com","username":"letuser","full_name":"Letters","password":"test123"}' | ConvertFrom-Json | Format-List
```

---

## 🎯 What to Verify

| Test Case | Expected Result |
|-----------|----------------|
| `9876543210` | ✅ Pass (normalized to +919876543210) |
| `+919876543210` | ✅ Pass |
| `98765-43210` | ✅ Pass (normalized) |
| `09876543210` | ✅ Pass (leading 0 removed) |
| `12345` | ❌ Fail (too short) |
| `abcdefghij` | ❌ Fail (not digits) |
| `+12345678901234567` | ❌ Fail (too many digits) |
| VoIP number (Twilio) | ❌ Fail (blocked) |
