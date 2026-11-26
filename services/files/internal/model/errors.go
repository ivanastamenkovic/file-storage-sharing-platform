package model

type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	ErrParentNotFound = &AppError{
		Status:  404,
		Code:    "PARENT_FOLDER_NOT_FOUND",
		Message: "Parent folder not found",
	}
	ErrFolderExists = &AppError{
		Status:  409,
		Code:    "FOLDER_ALREADY_EXISTS",
		Message: "A folder with this name already exists in the parent folder",
	}

	ErrUnauthorized = &AppError{
		Status:  401,
		Code:    "UNAUTHORIZED",
		Message: "Authentication required",
	}
	ErrInvalidRequest = &AppError{
		Status:  400,
		Code:    "INVALID_REQUEST",
		Message: "Invalid request body",
	}
)

func invalidName(msg string) *AppError {
	return &AppError{Status: 400, Code: "INVALID_FOLDER_NAME", Message: msg}
}
