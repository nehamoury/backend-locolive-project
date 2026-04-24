# Locolive Safety & Moderation Walkthrough

This document outlines the enhanced safety, trust, and moderation infrastructure implemented for production readiness.

## 1. Automated Trust Scoring System 🛡️
We have moved from a static trust value to a dynamic, real-time scoring algorithm.

### Algorithm Logic:
- **Base Score**: 50
- **Account Age**: +1 point per day (Max 20)
- **Social Connections**: +3 points per accepted connection (Max 15)
- **Community Reports**: -10 points per unresolved report (Max -30)
- **Verification**: +15 points for verified profiles

### Automation Hooks:
- **Connection Accepted**: Both users' trust scores are automatically recalculated.
- **Report Filed**: The target user's trust score is immediately recalculated.
- **Daily Growth**: Scores naturally improve as account age increases.

## 2. Advanced GPS Pattern Analysis 📍
To protect the integrity of proximity-based discovery, we've implemented sophisticated spoofing detection.

- **Grid Detection**: Detects bots moving in perfect straight lines or exact increments.
- **Jitter Check**: Identifies scripts that lack the natural "noise/jitter" of real hardware GPS.
- **Velocity Check**: Maintains existing protection against "impossible jumps" (Max 1000 km/h).

## 3. Real-time Content Moderation 🔍
Automated pre-screening is now active across all content entry points.

- **Text Filtering**: High-speed Regex patterns for profanity, scams, and spam.
- **Media Pre-screening**: A modular hook for AI Image/Video moderation (e.g., Google Vision/AWS Rekognition).
- **Instant Feedback**: Users receive a `MODERATION_FAILED` error immediately if they attempt to post toxic content.

## 4. Administrative Controls ⚙️
The admin layer is now fully synchronized with these new features.

- **Shadow Banning**: Admins can hide toxic users from the global map and feed.
- **Report Resolution**: Resolving a report can trigger a trust score recovery for the user.
- **Activity Logs**: Every moderation action is logged for accountability.

---
**Status**: The safety infrastructure is now production-hardened and feature-complete.
