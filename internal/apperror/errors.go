package apperror

import "errors"

var (
	ErrSegmentNotFound     = errors.New("segment not found")
	ErrSegmentConflict     = errors.New("segment cannot be in both 'add' and 'remove' lists")
	ErrUserSegmentNotFound = errors.New("user segment not found")

	ErrCannotInsertT  = errors.New("cannot insert into table")
	ErrCannotDeleteFT = errors.New("failed to remove from table")
	ErrCannotCreateT  = errors.New("cannot create table")

	ErrDuringRowsIteration = errors.New("error during rows iteration")

	ErrUserIDInvalid = errors.New("user id must be positive")
	ErrPercentAbove  = errors.New("percent must be less than 100")
	ErrPercentLess   = errors.New("percent must be above than 0")

	ErrSlugNotFound    = errors.New("slug not found")
	ErrEmptySlug       = errors.New("slug cannot be empty")
	ErrSlugLength      = errors.New("slug length must be between 3 and 50 characters")
	ErrSlugRegex       = errors.New("slug must contain only latin letters, digits, and underscores")
	ErrTooManySegments = errors.New("cannot update more than 100 segments in one request")

	ErrFailedBTransaction = errors.New("failed to begin transaction")
	ErrFailedCTransaction = errors.New("failed to commit transaction")
)
