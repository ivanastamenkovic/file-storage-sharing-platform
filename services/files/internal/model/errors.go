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
)

func invalidName(msg string) *AppError {
	return &AppError{Status: 400, Code: "INVALID_FOLDER_NAME", Message: msg}
}
