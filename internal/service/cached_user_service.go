package service

import (
	"context"

	"payflow/internal/domain"
	"payflow/pkg/cache"
	"payflow/pkg/logger"
)

// CachedUserService wraps UserService with caching functionality
type CachedUserService struct {
	userService  domain.UserService
	cache        cache.Cache
	cacheManager cache.CacheStrategy
	logger       logger.Logger
}

// NewCachedUserService creates a new cached user service
func NewCachedUserService(
	userService domain.UserService,
	cacheInstance cache.Cache,
	cacheManager cache.CacheStrategy,
	logger logger.Logger,
) domain.UserService {
	return &CachedUserService{
		userService:  userService,
		cache:        cacheInstance,
		cacheManager: cacheManager,
		logger:       logger,
	}
}

func (s *CachedUserService) GetUserByID(id int64) (*domain.User, error) {
	ctx := context.Background()
	key := cache.UserCacheKey(id)

	var user *domain.User
	err := s.cacheManager.ReadThrough(ctx, key, &user, func() (interface{}, error) {
		return s.userService.GetUserByID(id)
	}, cache.LongExpiration)

	if err != nil {
		s.logger.Error("Cache read-through error for user by ID", map[string]interface{}{
			"userID": id,
			"error":  err.Error(),
		})
		// Fallback to direct service call
		return s.userService.GetUserByID(id)
	}

	return user, nil
}

func (s *CachedUserService) GetUserByUsername(username string) (*domain.User, error) {
	ctx := context.Background()
	key := cache.UserCacheKeyByUsername(username)

	var user *domain.User
	err := s.cacheManager.ReadThrough(ctx, key, &user, func() (interface{}, error) {
		return s.userService.GetUserByUsername(username)
	}, cache.LongExpiration)

	if err != nil {
		s.logger.Error("Cache read-through error for user by username", map[string]interface{}{
			"username": username,
			"error":    err.Error(),
		})
		return s.userService.GetUserByUsername(username)
	}

	return user, nil
}

func (s *CachedUserService) GetUserByEmail(email string) (*domain.User, error) {
	ctx := context.Background()
	key := cache.UserCacheKeyByEmail(email)

	var user *domain.User
	err := s.cacheManager.ReadThrough(ctx, key, &user, func() (interface{}, error) {
		return s.userService.GetUserByEmail(email)
	}, cache.LongExpiration)

	if err != nil {
		s.logger.Error("Cache read-through error for user by email", map[string]interface{}{
			"email": email,
			"error": err.Error(),
		})
		return s.userService.GetUserByEmail(email)
	}

	return user, nil
}

func (s *CachedUserService) GetUserByApiKey(apiKey string) (*domain.User, error) {
	// API keys don't need caching as they're used for authentication
	return s.userService.GetUserByApiKey(apiKey)
}

func (s *CachedUserService) CreateUser(user *domain.User) error {
	ctx := context.Background()

	err := s.cacheManager.WriteThrough(ctx, cache.UserCacheKey(user.ID), user, func(value interface{}) error {
		return s.userService.CreateUser(user)
	}, cache.LongExpiration)

	if err != nil {
		s.logger.Error("Cache write-through error for user creation", map[string]interface{}{
			"userID": user.ID,
			"error":  err.Error(),
		})
		return s.userService.CreateUser(user)
	}

	// Cache by username and email as well
	if user.Username != "" {
		usernameKey := cache.UserCacheKeyByUsername(user.Username)
		if setErr := s.cache.Set(ctx, usernameKey, user, cache.LongExpiration); setErr != nil {
			s.logger.Error("Error caching user by username", map[string]interface{}{
				"username": user.Username,
				"error":    setErr.Error(),
			})
		}
	}

	if user.Email != "" {
		emailKey := cache.UserCacheKeyByEmail(user.Email)
		if setErr := s.cache.Set(ctx, emailKey, user, cache.LongExpiration); setErr != nil {
			s.logger.Error("Error caching user by email", map[string]interface{}{
				"email": user.Email,
				"error": setErr.Error(),
			})
		}
	}

	return nil
}

func (s *CachedUserService) UpdateUser(user *domain.User) error {
	ctx := context.Background()

	// Get old user data to invalidate old cache keys
	oldUser, _ := s.userService.GetUserByID(user.ID)

	err := s.cacheManager.WriteThrough(ctx, cache.UserCacheKey(user.ID), user, func(value interface{}) error {
		return s.userService.UpdateUser(user)
	}, cache.LongExpiration)

	if err != nil {
		s.logger.Error("Cache write-through error for user update", map[string]interface{}{
			"userID": user.ID,
			"error":  err.Error(),
		})
		return s.userService.UpdateUser(user)
	}

	// Invalidate old cache keys if username or email changed
	if oldUser != nil {
		if oldUser.Username != user.Username {
			oldUsernameKey := cache.UserCacheKeyByUsername(oldUser.Username)
			if delErr := s.cache.Delete(ctx, oldUsernameKey); delErr != nil {
				s.logger.Error("Error invalidating old username cache", map[string]interface{}{
					"oldUsername": oldUser.Username,
					"error":       delErr.Error(),
				})
			}
		}

		if oldUser.Email != user.Email {
			oldEmailKey := cache.UserCacheKeyByEmail(oldUser.Email)
			if delErr := s.cache.Delete(ctx, oldEmailKey); delErr != nil {
				s.logger.Error("Error invalidating old email cache", map[string]interface{}{
					"oldEmail": oldUser.Email,
					"error":    delErr.Error(),
				})
			}
		}
	}

	// Cache new data
	if user.Username != "" {
		usernameKey := cache.UserCacheKeyByUsername(user.Username)
		if setErr := s.cache.Set(ctx, usernameKey, user, cache.LongExpiration); setErr != nil {
			s.logger.Error("Error caching updated user by username", map[string]interface{}{
				"username": user.Username,
				"error":    setErr.Error(),
			})
		}
	}

	if user.Email != "" {
		emailKey := cache.UserCacheKeyByEmail(user.Email)
		if setErr := s.cache.Set(ctx, emailKey, user, cache.LongExpiration); setErr != nil {
			s.logger.Error("Error caching updated user by email", map[string]interface{}{
				"email": user.Email,
				"error": setErr.Error(),
			})
		}
	}

	return nil
}

func (s *CachedUserService) DeleteUser(id int64) error {
	ctx := context.Background()

	// Get user data to invalidate cache keys
	user, _ := s.userService.GetUserByID(id)

	err := s.userService.DeleteUser(id)
	if err != nil {
		return err
	}

	// Invalidate all user cache keys
	if cacheErr := cache.InvalidateUserCache(ctx, s.cache, id); cacheErr != nil {
		s.logger.Error("Error invalidating user cache", map[string]interface{}{
			"userID": id,
			"error":  cacheErr.Error(),
		})
	}

	// Invalidate by username and email if available
	if user != nil {
		if user.Username != "" {
			usernameKey := cache.UserCacheKeyByUsername(user.Username)
			if delErr := s.cache.Delete(ctx, usernameKey); delErr != nil {
				s.logger.Error("Error invalidating username cache", map[string]interface{}{
					"username": user.Username,
					"error":    delErr.Error(),
				})
			}
		}

		if user.Email != "" {
			emailKey := cache.UserCacheKeyByEmail(user.Email)
			if delErr := s.cache.Delete(ctx, emailKey); delErr != nil {
				s.logger.Error("Error invalidating email cache", map[string]interface{}{
					"email": user.Email,
					"error": delErr.Error(),
				})
			}
		}
	}

	return nil
}

func (s *CachedUserService) GenerateApiKey(userID int64) (string, error) {
	apiKey, err := s.userService.GenerateApiKey(userID)
	if err != nil {
		return "", err
	}

	// Invalidate user cache as API key has changed
	ctx := context.Background()
	if cacheErr := cache.InvalidateUserCache(ctx, s.cache, userID); cacheErr != nil {
		s.logger.Error("Error invalidating user cache after API key generation", map[string]interface{}{
			"userID": userID,
			"error":  cacheErr.Error(),
		})
	}

	return apiKey, nil
}

func (s *CachedUserService) HasAdminRole(userID int64) (bool, error) {
	// This could be cached but admin checks are usually not frequent enough to warrant caching
	return s.userService.HasAdminRole(userID)
}

func (s *CachedUserService) CheckPermission(userID int64, requiredRole string) (bool, error) {
	return s.userService.CheckPermission(userID, requiredRole)
}

func (s *CachedUserService) Login(username, password string) (string, error) {
	// Login should not be cached for security reasons
	return s.userService.Login(username, password)
}
