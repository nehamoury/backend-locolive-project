package api

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"time"
)

func (server *Server) setupRouter() {
	router := gin.New()

	// Global Middlewares
	router.Use(requestIDMiddleware())
	router.Use(loggerMiddleware())
	router.Use(gin.Recovery())

	// CORS Middleware
	router.Use(corsMiddleware(server.config.FrontendURL))

	// Security headers middleware
	router.Use(securityHeadersMiddleware(server.config.FrontendURL))

	// Enable gzip compression (70% bandwidth reduction)
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// Apply general rate limiting to all routes
	router.Use(server.generalRateLimiter())

	// Main API Group
	api := router.Group("/api")

	// Public routes inside API group
	api.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"status":  "ok",
			"message": "LocoLiv Backend is live!",
		})
	})
	api.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"status": "healthy",
		})
	})
	api.GET("/test-email", server.testEmail)
	api.POST("/users", server.authRateLimiter(), server.createUser)
	api.POST("/users/register", server.authRateLimiter(), server.createUser)
	api.POST("/users/login", server.authRateLimiter(), server.loginUser)
	api.POST("/users/renew-access", server.renewAccessToken)
	// Hard rate limit for logs to prevent DOS
	api.POST("/logs/client", server.rateLimitMiddleware(5, time.Minute), server.logClientError)

	// Username availability check (public with stricter rate limit)
	api.GET("/users/check-username", server.usernameCheckRateLimiter(), server.checkUsername)
	api.GET("/users/suggest-usernames", server.usernameCheckRateLimiter(), server.suggestUsernames)
	api.POST("/users/validate-username", server.usernameCheckRateLimiter(), server.validateUsername)
	api.GET("/users/check-email", server.usernameCheckRateLimiter(), server.checkEmail)
	api.GET("/users/check-phone", server.usernameCheckRateLimiter(), server.checkPhone)
	api.POST("/auth/google", server.authRateLimiter(), server.googleLogin)
	api.GET("/auth/google/callback", server.googleCallback)
	api.POST("/auth/forgot-password", server.authRateLimiter(), server.forgotPassword)
	api.POST("/auth/verify-reset-token", server.authRateLimiter(), server.verifyResetToken)
	api.POST("/auth/reset-password", server.authRateLimiter(), server.resetPassword)
	api.POST("/auth/verify-email", server.otpVerifyRateLimiter(), server.verifyEmail)

	// Protected routes (as sub-group of api)
	authRoutes := api.Group("/")
	authRoutes.Use(server.authMiddleware())

	// Profile completion (protected)
	authRoutes.POST("/users/complete-profile", server.completeProfile)
	authRoutes.POST("/account/set-password", server.rateLimitMiddleware(3, time.Hour), server.setPasswordForGoogleUser)

	// File upload
	authRoutes.POST("/upload", server.uploadFile)
	authRoutes.POST("/logout", server.logoutUser)
	authRoutes.POST("/auth/verify-firebase-phone", server.otpVerifyRateLimiter(), server.verifyFirebasePhone)
	authRoutes.POST("/auth/resend-email", server.otpResendRateLimiter(), server.resendEmailVerification)

	// Activation Gated Routes
	activeRoutes := authRoutes.Group("/")
	activeRoutes.Use(server.activationMiddleware())

	activeRoutes.POST("/location/ping", server.locationRateLimiter(), server.updateLocation)
	activeRoutes.GET("/location/heatmap", server.getHeatmap)
	// Stories
	activeRoutes.GET("/feed", server.getFeed)
	activeRoutes.POST("/stories", server.storyRateLimiter(), server.createStory)
	activeRoutes.GET("/stories/:id", server.getStory)
	activeRoutes.PUT("/stories/:id", server.updateStory)
	activeRoutes.DELETE("/stories/:id", server.deleteUserStory)
	activeRoutes.GET("/stories/map", server.getStoriesMap)
	activeRoutes.GET("/stories/connections", server.getConnectionStories)
	activeRoutes.GET("/stories/me", server.getMyStories)

	// Archive Stories
	activeRoutes.POST("/stories/:id/archive", server.archiveStory)
	activeRoutes.GET("/stories/archived", server.getArchivedStories)
	activeRoutes.DELETE("/stories/archived/:id", server.deleteArchivedStory)

	// --- CONNECTIONS / FOLLOWERS / FOLLOWING (PROPERLY STRUCTURED) ---
	// Current user endpoints (/me/*) - always for authenticated user
	activeRoutes.GET("/me/connections", server.listMeConnections)
	activeRoutes.GET("/me/followers", server.listMeFollowers)
	activeRoutes.GET("/me/following", server.listMeFollowing)

	// Target user endpoints (/users/:id/*) - for viewing other profiles with privacy checks
	activeRoutes.GET("/users/:id/connections", server.listUserConnections)
	activeRoutes.GET("/users/:id/followers", server.listUserFollowers)
	activeRoutes.GET("/users/:id/following", server.listUserFollowing)

	// Legacy endpoints (kept for backward compatibility during transition)
	activeRoutes.GET("/connections", server.listMeConnections)
	activeRoutes.GET("/connections/followers", server.listMeFollowers)
	activeRoutes.GET("/connections/following", server.listMeFollowing)

	// Other connection-related endpoints
	activeRoutes.GET("/connections/suggested", server.getSuggestedConnections)
	activeRoutes.GET("/connections/requests", server.listPendingRequests)
	activeRoutes.GET("/connections/sent", server.listSentRequests)
	activeRoutes.POST("/connections/request", server.engagementRateLimiter(), server.sendConnectionRequest)
	activeRoutes.POST("/connections/update", server.engagementRateLimiter(), server.updateConnection)
	activeRoutes.DELETE("/connections/:id", server.engagementRateLimiter(), server.deleteConnection)

	// Notifications
	activeRoutes.GET("/notifications", server.getNotifications)
	activeRoutes.PUT("/notifications/:id/read", server.markNotificationRead)
	activeRoutes.PUT("/notifications/read-all", server.markAllNotificationsRead)
	activeRoutes.DELETE("/notifications/:id", server.deleteNotification)
	activeRoutes.DELETE("/notifications", server.deleteAllNotifications)
	activeRoutes.GET("/notifications/unread-count", server.getUnreadCount)
	activeRoutes.POST("/notifications/token", server.registerFCMToken)
	activeRoutes.DELETE("/notifications/token", server.removeFCMToken)

	// Chat & Messages
	activeRoutes.GET("/conversations", server.getConversationList)
	activeRoutes.GET("/messages", server.messageRateLimiter(), server.getChatHistory)
	activeRoutes.POST("/messages", server.messageRateLimiter(), server.sendMessage)
	activeRoutes.GET("/messages/unread-count", server.getUnreadMessageCount)
	activeRoutes.PUT("/messages/read/:userId", server.markConversationRead)
	activeRoutes.DELETE("/messages/:id", server.deleteMessage)
	activeRoutes.PUT("/messages/:id", server.editMessage)
	activeRoutes.PUT("/messages/:id/save", server.saveMessage)
	activeRoutes.DELETE("/conversations/:userId", server.deleteConversation)
	activeRoutes.POST("/messages/:id/reactions", server.addReaction)
	activeRoutes.DELETE("/messages/:id/reactions", server.removeReaction)
	activeRoutes.GET("/messages/:id/reactions", server.getMessageReactions)
	activeRoutes.GET("/chat/icebreakers", server.getIcebreakers)
	activeRoutes.GET("/ws/chat", server.chatWebSocket)

	activeRoutes.GET("/crossings", server.getCrossings)
	authRoutes.PUT("/profile", server.updateProfile)
	activeRoutes.POST("/reports", server.createReport)
	activeRoutes.POST("/profile/boost", server.boostProfile)
	authRoutes.PUT("/account/email", server.rateLimitMiddleware(3, time.Hour), server.updateUserEmail)
	authRoutes.PUT("/account/password", server.rateLimitMiddleware(3, time.Hour), server.updateUserPassword)
	authRoutes.POST("/account/verify-password", server.rateLimitMiddleware(10, time.Minute), server.verifyPassword)
	authRoutes.POST("/account/logout-all", server.logoutAllDevices)
	authRoutes.DELETE("/account", server.deleteAccount)

	// User Preferences & Settings
	authRoutes.GET("/settings/preferences", server.getPreferences)
	authRoutes.PUT("/settings/preferences", server.updatePreferences)
	authRoutes.GET("/settings/notifications", server.getNotificationSettings)
	authRoutes.PUT("/settings/notifications", server.updateNotificationSettings)

	// Support System
	authRoutes.POST("/support/tickets", server.rateLimitMiddleware(5, 24*time.Hour), server.createSupportTicket)
	authRoutes.GET("/support/tickets", server.listMySupportTickets)

	// Data & Privacy
	authRoutes.POST("/account/data-export", server.requestDataExport)
	authRoutes.GET("/account/data-export", server.getDataExportStatus)

	// Username Management
	authRoutes.POST("/users/reserve-username", server.reserveUsername)
	authRoutes.PUT("/users/change-username", server.changeUsername)

	// Gamification & Stats
	activeRoutes.GET("/stats/streak", server.getStreak)
	activeRoutes.GET("/stats/daily", server.getDailyStats)
	activeRoutes.GET("/badges", server.listBadges)
	authRoutes.PUT("/notifications/preferences", server.updateNotificationPreferences)
	authRoutes.GET("/notifications/preferences", server.getNotificationPreferences)

	// Privacy features
	authRoutes.GET("/privacy", server.getPrivacySettings)
	authRoutes.PUT("/privacy", server.updatePrivacySettings)
	authRoutes.PATCH("/user/privacy", server.updateAccountPrivacy)
	activeRoutes.POST("/users/block", server.blockUser)
	activeRoutes.DELETE("/users/block/:id", server.unblockUser)
	activeRoutes.GET("/users/blocked", server.getBlockedUsers)
	activeRoutes.PUT("/location/ghost-mode", server.toggleGhostMode) // Keep legacy for compatibility if any
	activeRoutes.POST("/users/ghost-mode", server.toggleGhostMode)
	activeRoutes.POST("/users/panic", server.panicMode)

	// Story engagement
	activeRoutes.POST("/stories/:id/view", server.viewStory)
	activeRoutes.GET("/stories/:id/viewers", server.getStoryViewers)
	activeRoutes.POST("/stories/:id/react", server.reactToStory)
	activeRoutes.DELETE("/stories/:id/react", server.deleteStoryReaction)
	activeRoutes.GET("/stories/:id/reactions", server.getStoryReactions)
	activeRoutes.POST("/stories/share", server.shareStory)

	// Activity & Visibility
	activeRoutes.GET("/activity/status", server.getActivityStatus)

	// User Profiles
	activeRoutes.GET("/users/search", server.searchRateLimiter(), server.searchUsers)
	activeRoutes.GET("/users/nearby", server.getNearbyUsers)
	activeRoutes.GET("/users/:id", server.privacyCheckMiddleware(), server.getUserProfile)
	activeRoutes.GET("/stories/user/:id", server.privacyCheckMiddleware(), server.getUserStories)
	authRoutes.GET("/profile/me", server.getMyProfile)
	activeRoutes.GET("/profile/visitors", server.getProfileVisitors)

	// Posts
	activeRoutes.POST("/posts", server.createPost)
	activeRoutes.GET("/posts/feed", server.getConnectionsFeed)
	activeRoutes.GET("/posts/me", server.getMyPosts)
	activeRoutes.GET("/posts/saved", server.getSavedPosts)
	activeRoutes.GET("/users/:id/posts", server.privacyCheckMiddleware(), server.getUserPosts)
	activeRoutes.DELETE("/posts/:id", server.deletePost)
	activeRoutes.POST("/posts/:id/like", server.engagementRateLimiter(), server.likePost)
	activeRoutes.DELETE("/posts/:id/like", server.engagementRateLimiter(), server.unlikePost)
	activeRoutes.GET("/posts/:id/comments", server.listPostComments)
	activeRoutes.POST("/posts/:id/comments", server.addPostComment)
	activeRoutes.POST("/posts/:id/share", server.engagementRateLimiter(), server.sharePost)
	activeRoutes.POST("/posts/:id/save", server.engagementRateLimiter(), server.savePost)
	activeRoutes.DELETE("/posts/:id/save", server.engagementRateLimiter(), server.unsavePost)
	activeRoutes.DELETE("/posts/:id/comments/:commentId", server.deletePostComment)

	// Reels
	activeRoutes.POST("/reels", server.createReel)
	activeRoutes.GET("/reels/feed", server.getReelsFeed)
	activeRoutes.GET("/reels/me", server.getMyReels)
	activeRoutes.GET("/reels/nearby", server.getNearbyReels)
	activeRoutes.GET("/reels/saved", server.getSavedReels)
	activeRoutes.GET("/users/:id/reels", server.privacyCheckMiddleware(), server.getUserReels)
	activeRoutes.DELETE("/reels/:id", server.deleteReel)
	activeRoutes.POST("/reels/:id/like", server.engagementRateLimiter(), server.likeReel)
	activeRoutes.DELETE("/reels/:id/like", server.engagementRateLimiter(), server.unlikeReel)
	activeRoutes.POST("/reels/:id/comments", server.addReelComment)
	activeRoutes.GET("/reels/:id/comments", server.listReelComments)
	activeRoutes.POST("/reels/:id/share", server.engagementRateLimiter(), server.shareReel)
	activeRoutes.POST("/reels/:id/save", server.engagementRateLimiter(), server.saveReel)
	activeRoutes.DELETE("/reels/:id/save", server.engagementRateLimiter(), server.unsaveReel)
	activeRoutes.DELETE("/reels/:id/comments/:commentId", server.deleteReelComment)

	// Highlights
	activeRoutes.POST("/highlights", server.createHighlight)
	activeRoutes.GET("/highlights/me", server.getMyHighlights)
	activeRoutes.GET("/users/:id/highlights", server.privacyCheckMiddleware(), server.getHighlights)
	activeRoutes.GET("/highlights/:id", server.getHighlightDetails)
	activeRoutes.POST("/highlights/:id/stories", server.addStoryToHighlight)
	activeRoutes.DELETE("/highlights/:id/stories/:storyId", server.removeStoryFromHighlight)
	activeRoutes.DELETE("/highlights/:id", server.deleteHighlight)

	// Groups
	activeRoutes.POST("/groups", server.createGroup)
	activeRoutes.GET("/groups", server.getMyGroups)
	activeRoutes.GET("/groups/:id/messages", server.getGroupMessages)

	// Admin routes
	adminRoutes := api.Group("/admin")
	adminRoutes.Use(server.authMiddleware())
	adminRoutes.Use(adminMiddleware())

	// A. Dashboard
	adminRoutes.GET("/dashboard", server.getAdminDashboard)

	// B. User Management
	adminRoutes.GET("/users", server.listUsers)
	adminRoutes.GET("/users/:id", server.getAdminUserDetail)
	adminRoutes.POST("/users/:id/actions", server.handleAdminUserAction)
	adminRoutes.GET("/users/search", server.searchUsersAdmin) // Keep for legacy/compat

	// C. Content Moderation
	adminRoutes.GET("/content", server.listAdminContent)
	adminRoutes.DELETE("/stories/:id", server.deleteStory)
	adminRoutes.DELETE("/posts/:id", server.deletePost) // Admin override
	adminRoutes.DELETE("/reels/:id", server.deleteReel) // Admin override

	// D. Reports System
	adminRoutes.GET("/reports", server.listReports)
	adminRoutes.PUT("/reports/:id/resolve", server.resolveReport)

	// E. Blocks / Privacy
	adminRoutes.GET("/blocks", server.listAdminBlocks)

	// F. Engagement Inspector
	adminRoutes.GET("/engagement", server.inspectEngagement)

	// G. Logs / Error Viewer
	adminRoutes.GET("/logs", server.listAdminLogs)
	adminRoutes.GET("/activity/logs", server.listActivityLogs) // Legacy

	// H. System Monitor
	adminRoutes.GET("/system", server.getSystemMonitor)

	// Existing Admin Tools (Keep or adapt)
	adminRoutes.GET("/activity", server.activityWebSocket)
	adminRoutes.GET("/comments", server.listAllComments)
	adminRoutes.POST("/comments/moderate", server.moderateComment)
	adminRoutes.GET("/map/active", server.getMapActiveUsers)
	adminRoutes.GET("/crossings", server.listAdminCrossings)
	adminRoutes.POST("/notifications/send", server.sendBroadcastNotification)
	adminRoutes.GET("/notifications", server.listAdminNotifications)
	adminRoutes.GET("/settings", server.getAppSettings)
	adminRoutes.PUT("/settings", server.updateAppSettings)

	// Admin Users CRUD
	adminRoutes.GET("/admins", server.listAdminUsers)
	adminRoutes.POST("/admins", server.createAdminUser)
	adminRoutes.PUT("/admins/:id", server.updateAdminUser)
	adminRoutes.DELETE("/admins/:id", server.deleteAdminUser)

	// Reserved Username Management
	adminRoutes.GET("/reserved-usernames", server.listReservedUsernames)
	adminRoutes.POST("/reserved-usernames", server.addReservedUsername)
	adminRoutes.DELETE("/reserved-usernames/:username", server.removeReservedUsername)

	// Serve uploaded media files with 1-year cache
	router.Static("/uploads", "./uploads")
	router.Group("/uploads").Use(func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	})

	// Frontend static files with cache
	assets := router.Group("/assets")
	assets.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	})
	assets.Static("", "../../frontend/frontend/dist/assets")

	router.StaticFile("/manifest.webmanifest", "../../frontend/frontend/dist/manifest.webmanifest")
	router.StaticFile("/sw.js", "../../frontend/frontend/dist/sw.js")
	router.StaticFile("/pwa-192x192.png", "../../frontend/frontend/dist/pwa-192x192.png")
	router.StaticFile("/pwa-512x512.png", "../../frontend/frontend/dist/pwa-512x512.png")
	router.StaticFile("/favicon.svg", "../../frontend/frontend/dist/favicon.svg")
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})

	// SPA fallback: serve index.html - NEVER CACHE THIS
	router.NoRoute(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		c.File("../../frontend/frontend/dist/index.html")
	})

	server.router = router
}
