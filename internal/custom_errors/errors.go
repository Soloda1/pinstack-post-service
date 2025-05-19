package custom_errors

import "errors"

// Ошибки пользователя
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUsernameExists    = errors.New("username already exists")
	ErrEmailExists       = errors.New("email already exists")
	ErrInvalidUsername   = errors.New("invalid username")
	ErrInvalidEmail      = errors.New("invalid email")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrPasswordMismatch  = errors.New("passwords do not match")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// Ошибки валидации
var (
	ErrPostValidation   = errors.New("post validation failed")
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidInput     = errors.New("invalid input")
	ErrRequiredField    = errors.New("required field is missing")
)

// Ошибки базы данных
var (
	ErrDatabaseConnection  = errors.New("database connection error")
	ErrDatabaseQuery       = errors.New("database query error")
	ErrDatabaseTransaction = errors.New("database transaction error")
	ErrDatabaseScan        = errors.New("database scan error")
)

// Ошибки внешних сервисов
var (
	ErrExternalServiceUnavailable = errors.New("external service unavailable")
	ErrExternalServiceTimeout     = errors.New("external service timeout")
	ErrExternalServiceError       = errors.New("external service error")
)

// Ошибки файловой системы
var (
	ErrFileNotFound     = errors.New("file not found")
	ErrFileAccessDenied = errors.New("file access denied")
	ErrFileTooLarge     = errors.New("file too large")
)

// Ошибки конфигурации
var (
	ErrConfigNotFound   = errors.New("configuration not found")
	ErrConfigInvalid    = errors.New("invalid configuration")
	ErrConfigLoadFailed = errors.New("failed to load configuration")
)

// Ошибки кэша
var (
	ErrCacheMiss     = errors.New("cache miss")
	ErrCacheDisabled = errors.New("cache disabled")
	ErrCacheError    = errors.New("cache error")
)

// Ошибки rate limiting
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrTooManyRequests   = errors.New("too many requests")
)

// Ошибки бизнес-логики
var (
	ErrOperationNotAllowed = errors.New("operation not allowed")
	ErrResourceLocked      = errors.New("resource is locked")
	ErrInsufficientRights  = errors.New("insufficient rights")
)

// Ошибки поиска
var (
	ErrSearchFailed       = errors.New("search failed")
	ErrInvalidSearchQuery = errors.New("invalid search query")
)

// Ошибки внутренних сервисов
var (
	ErrInternalServiceError = errors.New("internal service error")
)

// Ошибки аватара
var (
	ErrInvalidAvatarFormat = errors.New("invalid avatar format")
	ErrAvatarUploadFailed  = errors.New("avatar upload failed")
	ErrAvatarDeleteFailed  = errors.New("avatar delete failed")
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Ошибки постов
var (
	ErrNoUpdateRows = errors.New("no post update rows")
	ErrPostNotFound = errors.New("post not found")
)

// Ошибки тегов
var (
	ErrTagsNotFound        = errors.New("tags not found")
	ErrTagNotFound         = errors.New("tag not found")
	ErrInvalidTagName      = errors.New("invalid tag name")
	ErrTagAlreadyExists    = errors.New("tag already exists")
	ErrTagPost             = errors.New("failed to tag post")
	ErrTagQueryFailed      = errors.New("failed to query tags")
	ErrTagScanFailed       = errors.New("failed to scan tag row")
	ErrTagCreateFailed     = errors.New("failed to create tag")
	ErrTagDeleteFailed     = errors.New("failed to delete tag")
	ErrTagUntagFailed      = errors.New("failed to untag post")
	ErrTagInsertFailed     = errors.New("failed to insert tag relation")
	ErrTagVerifyPostFailed = errors.New("failed to verify post for tag operation")
)

// Media operations
var (
	ErrMediaRemoveFailed     = errors.New("media remove failed")
	ErrMediaNotFound         = errors.New("media not found")
	ErrMediaAttachFailed     = errors.New("failed to attach media to post")
	ErrMediaReorderFailed    = errors.New("failed to reorder media positions")
	ErrMediaDetachFailed     = errors.New("failed to detach media from post")
	ErrMediaQueryFailed      = errors.New("failed to query post media")
	ErrMediaBatchQueryFailed = errors.New("failed to batch query posts media")
)
