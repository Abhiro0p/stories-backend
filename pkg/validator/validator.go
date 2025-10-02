package validator

import (
    "fmt"
    "regexp"
    "unicode"

    "github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
    validate = validator.New()
    
    // Register custom validators
    validate.RegisterValidation("username", validateUsername)
    validate.RegisterValidation("strong_password", validateStrongPassword)
    validate.RegisterValidation("story_text", validateStoryText)
    validate.RegisterValidation("visibility", validateVisibility)
    validate.RegisterValidation("reaction_type", validateReactionType)
}

// ValidateStruct validates a struct using validator tags
func ValidateStruct(s interface{}) error {
    return validate.Struct(s)
}

// validateUsername validates username format
func validateUsername(fl validator.FieldLevel) bool {
    username := fl.Field().String()
    
    if len(username) < 3 || len(username) > 30 {
        return false
    }
    
    // Username should contain only alphanumeric characters and underscores
    matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", username)
    return matched
}

// validateStrongPassword validates password strength
func validateStrongPassword(fl validator.FieldLevel) bool {
    password := fl.Field().String()
    
    if len(password) < 8 {
        return false
    }
    
    var hasUpper, hasLower, hasNumber, hasSymbol bool
    
    for _, char := range password {
        switch {
        case unicode.IsUpper(char):
            hasUpper = true
        case unicode.IsLower(char):
            hasLower = true
        case unicode.IsDigit(char):
            hasNumber = true
        case unicode.IsPunct(char) || unicode.IsSymbol(char):
            hasSymbol = true
        }
    }
    
    // Require at least 3 out of 4 character types
    count := 0
    if hasUpper { count++ }
    if hasLower { count++ }
    if hasNumber { count++ }
    if hasSymbol { count++ }
    
    return count >= 3
}

// validateStoryText validates story text content
func validateStoryText(fl validator.FieldLevel) bool {
    text := fl.Field().String()
    return len(text) <= 2000 && len(text) > 0
}

// validateVisibility validates story visibility values
func validateVisibility(fl validator.FieldLevel) bool {
    visibility := fl.Field().String()
    validValues := []string{"public", "private", "friends"}
    
    for _, valid := range validValues {
        if visibility == valid {
            return true
        }
    }
    
    return false
}

// validateReactionType validates reaction type values
func validateReactionType(fl validator.FieldLevel) bool {
    reactionType := fl.Field().String()
    validTypes := []string{"like", "love", "laugh", "wow", "sad", "angry", "fire", "hundred"}
    
    for _, valid := range validTypes {
        if reactionType == valid {
            return true
        }
    }
    
    return false
}
